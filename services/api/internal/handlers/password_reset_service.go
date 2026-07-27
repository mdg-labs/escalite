package handlers

import (
	"context"
	"errors"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mdg-labs/escalite/services/api/internal/auth"
	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/api/internal/email"
	"github.com/mdg-labs/escalite/services/api/internal/ratelimit"
)

const defaultResetTokenTTL = time.Hour

// PasswordResetAcceptedMessage is returned for successful reset requests regardless of account existence.
const PasswordResetAcceptedMessage = "if an account exists, a password reset email has been sent"

// PasswordResetService implements password reset request and confirm flows.
type PasswordResetService struct {
	pool    *pgxpool.Pool
	queries db.Querier
	mail    email.Sender
	cfg     PasswordResetConfig
	logger  interface {
		Error(msg string, args ...any)
	}
}

// NewPasswordResetService returns a password reset service.
func NewPasswordResetService(
	pool *pgxpool.Pool,
	queries db.Querier,
	mail email.Sender,
	cfg PasswordResetConfig,
	logger interface {
		Error(msg string, args ...any)
	},
) *PasswordResetService {
	if cfg.ResetTokenTTL <= 0 {
		cfg.ResetTokenTTL = defaultResetTokenTTL
	}
	return &PasswordResetService{
		pool:    pool,
		queries: queries,
		mail:    mail,
		cfg:     cfg,
		logger:  logger,
	}
}

// RequestReset validates input, applies rate limits, and issues a reset token when an account exists.
func (s *PasswordResetService) RequestReset(ctx context.Context, emailAddr, clientIP string) (string, error) {
	emailAddr = strings.ToLower(strings.TrimSpace(emailAddr))
	if emailAddr == "" {
		return "", &PasswordResetError{Code: CodeValidation, Message: "email is required"}
	}
	if _, err := mail.ParseAddress(emailAddr); err != nil {
		return "", &PasswordResetError{Code: CodeValidation, Message: "email is invalid"}
	}

	clientIP = strings.TrimSpace(clientIP)
	if s.cfg.EmailLimiter != nil && !s.cfg.EmailLimiter.Allow("email:"+emailAddr) {
		return "", &PasswordResetError{Code: CodeRateLimited, Message: "too many password reset requests for this email"}
	}
	if s.cfg.IPLimiter != nil && clientIP != "" && !s.cfg.IPLimiter.Allow("ip:"+clientIP) {
		return "", &PasswordResetError{Code: CodeRateLimited, Message: "too many password reset requests from this IP"}
	}

	account, err := s.queries.GetAccountByEmail(ctx, emailAddr)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return PasswordResetAcceptedMessage, nil
		}
		s.logger.Error("password reset account lookup failed", "error", err)
		return "", &PasswordResetError{Code: CodeInternal, Message: "internal error"}
	}

	user, err := auth.LoginMembership(ctx, s.queries, account.ID)
	if err != nil {
		if errors.Is(err, auth.ErrNoActiveMembership) {
			return PasswordResetAcceptedMessage, nil
		}
		s.logger.Error("password reset membership lookup failed", "error", err)
		return "", &PasswordResetError{Code: CodeInternal, Message: "internal error"}
	}

	if err := s.issueResetToken(ctx, user, emailAddr); err != nil {
		s.logger.Error("password reset token issue failed", "error", err)
		return "", &PasswordResetError{Code: CodeInternal, Message: "internal error"}
	}

	return PasswordResetAcceptedMessage, nil
}

// ConfirmReset validates a reset token and updates the account password.
func (s *PasswordResetService) ConfirmReset(ctx context.Context, token, password string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return &PasswordResetError{Code: CodeValidation, Message: "token is required"}
	}
	if len(password) < minPasswordLength {
		return &PasswordResetError{Code: CodeValidation, Message: "password must be at least 8 characters"}
	}

	tokenHash := auth.HashPasswordResetToken(token)
	resetToken, err := s.queries.GetPasswordResetTokenByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &PasswordResetError{Code: CodeValidation, Message: "reset token is invalid or expired"}
		}
		s.logger.Error("password reset token lookup failed", "error", err)
		return &PasswordResetError{Code: CodeInternal, Message: "internal error"}
	}

	if resetToken.UsedAt.Valid {
		return &PasswordResetError{Code: CodeValidation, Message: "reset token is invalid or expired"}
	}
	if !resetToken.ExpiresAt.Valid || !resetToken.ExpiresAt.Time.After(time.Now().UTC()) {
		return &PasswordResetError{Code: CodeValidation, Message: "reset token is invalid or expired"}
	}

	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		s.logger.Error("hash password failed", "error", err)
		return &PasswordResetError{Code: CodeInternal, Message: "internal error"}
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		s.logger.Error("begin transaction failed", "error", err)
		return &PasswordResetError{Code: CodeInternal, Message: "internal error"}
	}
	defer func() { _ = tx.Rollback(ctx) }()

	txQueries := db.New(s.pool).WithTx(tx)

	resetUser, err := txQueries.GetUserByID(ctx, db.GetUserByIDParams{
		ID:             resetToken.UserID,
		OrganizationID: resetToken.OrganizationID,
	})
	if err != nil {
		s.logger.Error("load reset user failed", "error", err)
		return &PasswordResetError{Code: CodeInternal, Message: "internal error"}
	}

	if err := txQueries.UpdateAccountPasswordHash(ctx, db.UpdateAccountPasswordHashParams{
		ID:           resetUser.AccountID,
		PasswordHash: pgtype.Text{String: passwordHash, Valid: true},
	}); err != nil {
		s.logger.Error("update password failed", "error", err)
		return &PasswordResetError{Code: CodeInternal, Message: "internal error"}
	}

	if err := txQueries.MarkPasswordResetTokenUsed(ctx, resetToken.ID); err != nil {
		s.logger.Error("mark reset token used failed", "error", err)
		return &PasswordResetError{Code: CodeInternal, Message: "internal error"}
	}

	if err := tx.Commit(ctx); err != nil {
		s.logger.Error("commit password reset failed", "error", err)
		return &PasswordResetError{Code: CodeInternal, Message: "internal error"}
	}

	return nil
}

// PasswordResetError is a coded password reset failure.
type PasswordResetError struct {
	Code    string
	Message string
}

func (e *PasswordResetError) Error() string {
	return e.Message
}

func (s *PasswordResetService) issueResetToken(ctx context.Context, user db.User, emailAddr string) error {
	plaintext, tokenHash, err := auth.NewPasswordResetToken()
	if err != nil {
		return err
	}

	if err := s.queries.InvalidateUnusedPasswordResetTokensForUser(ctx, user.ID); err != nil {
		return err
	}

	expiresAt := time.Now().UTC().Add(s.cfg.ResetTokenTTL)
	_, err = s.queries.CreatePasswordResetToken(ctx, db.CreatePasswordResetTokenParams{
		ID:             uuid.Must(uuid.NewV7()),
		UserID:         user.ID,
		OrganizationID: user.OrganizationID,
		TokenHash:      tokenHash,
		ExpiresAt:      pgtype.Timestamptz{Time: expiresAt, Valid: true},
	})
	if err != nil {
		return err
	}

	resetURL := buildPasswordResetURL(s.cfg.PublicURL, plaintext)
	body := "Use the link below to reset your Escalite password. The link expires in one hour.\n\n" + resetURL + "\n"
	if resetURL == plaintext {
		body = "Use this token to reset your Escalite password. The token expires in one hour.\n\n" + plaintext + "\n"
	}

	return s.mail.Send(ctx, email.Message{
		To:      emailAddr,
		Subject: "Reset your Escalite password",
		Body:    body,
	})
}

func buildPasswordResetURL(publicURL, token string) string {
	base := strings.TrimSuffix(strings.TrimSpace(publicURL), "/")
	if base == "" {
		return token
	}
	return base + "/reset-password?token=" + token
}

// PasswordResetConfig controls password reset behavior.
type PasswordResetConfig struct {
	PublicURL     string
	ResetTokenTTL time.Duration
	EmailLimiter  ratelimit.Limiter
	IPLimiter     ratelimit.Limiter
}
