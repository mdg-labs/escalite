package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
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

const (
	passwordResetAcceptedMessage = "if an account exists, a password reset email has been sent"
	defaultResetTokenTTL         = time.Hour
)

// PasswordResetConfig controls password reset behavior.
type PasswordResetConfig struct {
	PublicURL      string
	ResetTokenTTL  time.Duration
	EmailLimiter   ratelimit.Limiter
	IPLimiter      ratelimit.Limiter
}

// PasswordResetHandler handles password reset request and confirm endpoints.
type PasswordResetHandler struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
	mail   email.Sender
	cfg    PasswordResetConfig
}

// NewPasswordResetHandler returns handlers for password reset flows.
func NewPasswordResetHandler(
	pool *pgxpool.Pool,
	logger *slog.Logger,
	mail email.Sender,
	cfg PasswordResetConfig,
) *PasswordResetHandler {
	if cfg.ResetTokenTTL <= 0 {
		cfg.ResetTokenTTL = defaultResetTokenTTL
	}
	return &PasswordResetHandler{
		pool:   pool,
		logger: logger,
		mail:   mail,
		cfg:    cfg,
	}
}

type passwordResetRequestBody struct {
	Email string `json:"email"`
}

type passwordResetConfirmBody struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

type passwordResetAcceptedResponse struct {
	Message string `json:"message"`
}

// Request handles POST /api/v1/password-reset/request.
func (h *PasswordResetHandler) Request(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteAPIError(w, http.StatusMethodNotAllowed, CodeMethodNotAllowed, "method not allowed")
		return
	}

	var req passwordResetRequestBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteAPIError(w, http.StatusBadRequest, CodeValidation, "invalid JSON body")
		return
	}

	emailAddr := strings.ToLower(strings.TrimSpace(req.Email))
	if emailAddr == "" {
		WriteAPIError(w, http.StatusBadRequest, CodeValidation, "email is required")
		return
	}
	if _, err := mail.ParseAddress(emailAddr); err != nil {
		WriteAPIError(w, http.StatusBadRequest, CodeValidation, "email is invalid")
		return
	}

	clientIP := strings.TrimSpace(r.RemoteAddr)
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
		clientIP = strings.TrimSpace(strings.Split(forwarded, ",")[0])
	}

	if h.cfg.EmailLimiter != nil && !h.cfg.EmailLimiter.Allow("email:"+emailAddr) {
		WriteAPIError(w, http.StatusTooManyRequests, CodeRateLimited, "too many password reset requests for this email")
		return
	}
	if h.cfg.IPLimiter != nil && clientIP != "" && !h.cfg.IPLimiter.Allow("ip:"+clientIP) {
		WriteAPIError(w, http.StatusTooManyRequests, CodeRateLimited, "too many password reset requests from this IP")
		return
	}

	ctx := r.Context()
	queries := db.New(h.pool)

	user, err := queries.GetUserByEmailForAuth(ctx, emailAddr)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			h.writeAccepted(w)
			return
		}
		h.logger.Error("password reset user lookup failed", "error", err)
		WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
		return
	}

	if err := h.issueResetToken(ctx, queries, user, emailAddr); err != nil {
		h.logger.Error("password reset token issue failed", "error", err)
		WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
		return
	}

	h.writeAccepted(w)
}

// Confirm handles POST /api/v1/password-reset/confirm.
func (h *PasswordResetHandler) Confirm(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteAPIError(w, http.StatusMethodNotAllowed, CodeMethodNotAllowed, "method not allowed")
		return
	}

	var req passwordResetConfirmBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteAPIError(w, http.StatusBadRequest, CodeValidation, "invalid JSON body")
		return
	}

	token := strings.TrimSpace(req.Token)
	password := req.Password

	if token == "" {
		WriteAPIError(w, http.StatusBadRequest, CodeValidation, "token is required")
		return
	}
	if len(password) < minPasswordLength {
		WriteAPIError(w, http.StatusBadRequest, CodeValidation, "password must be at least 8 characters")
		return
	}

	ctx := r.Context()
	queries := db.New(h.pool)

	tokenHash := auth.HashPasswordResetToken(token)
	resetToken, err := queries.GetPasswordResetTokenByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteAPIError(w, http.StatusBadRequest, CodeValidation, "reset token is invalid or expired")
			return
		}
		h.logger.Error("password reset token lookup failed", "error", err)
		WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
		return
	}

	if resetToken.UsedAt.Valid {
		WriteAPIError(w, http.StatusBadRequest, CodeValidation, "reset token is invalid or expired")
		return
	}
	if !resetToken.ExpiresAt.Valid || !resetToken.ExpiresAt.Time.After(time.Now().UTC()) {
		WriteAPIError(w, http.StatusBadRequest, CodeValidation, "reset token is invalid or expired")
		return
	}

	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		h.logger.Error("hash password failed", "error", err)
		WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
		return
	}

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		h.logger.Error("begin transaction failed", "error", err)
		WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	txQueries := queries.WithTx(tx)

	if err := txQueries.UpdateUserPasswordHash(ctx, db.UpdateUserPasswordHashParams{
		ID:           resetToken.UserID,
		PasswordHash: pgtype.Text{String: passwordHash, Valid: true},
	}); err != nil {
		h.logger.Error("update password failed", "error", err)
		WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
		return
	}

	if err := txQueries.MarkPasswordResetTokenUsed(ctx, resetToken.ID); err != nil {
		h.logger.Error("mark reset token used failed", "error", err)
		WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
		return
	}

	if err := tx.Commit(ctx); err != nil {
		h.logger.Error("commit password reset failed", "error", err)
		WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *PasswordResetHandler) issueResetToken(ctx context.Context, queries *db.Queries, user db.User, emailAddr string) error {
	plaintext, tokenHash, err := auth.NewPasswordResetToken()
	if err != nil {
		return err
	}

	if err := queries.InvalidateUnusedPasswordResetTokensForUser(ctx, user.ID); err != nil {
		return err
	}

	expiresAt := time.Now().UTC().Add(h.cfg.ResetTokenTTL)
	_, err = queries.CreatePasswordResetToken(ctx, db.CreatePasswordResetTokenParams{
		ID:             uuid.Must(uuid.NewV7()),
		UserID:         user.ID,
		OrganizationID: user.OrganizationID,
		TokenHash:      tokenHash,
		ExpiresAt:      pgtype.Timestamptz{Time: expiresAt, Valid: true},
	})
	if err != nil {
		return err
	}

	resetURL := h.buildResetURL(plaintext)
	body := "Use the link below to reset your Escalite password. The link expires in one hour.\n\n" + resetURL + "\n"
	if resetURL == plaintext {
		body = "Use this token to reset your Escalite password. The token expires in one hour.\n\n" + plaintext + "\n"
	}

	return h.mail.Send(ctx, email.Message{
		To:      emailAddr,
		Subject: "Reset your Escalite password",
		Body:    body,
	})
}

func (h *PasswordResetHandler) buildResetURL(token string) string {
	base := strings.TrimSuffix(strings.TrimSpace(h.cfg.PublicURL), "/")
	if base == "" {
		return token
	}
	return base + "/reset-password?token=" + token
}

func (h *PasswordResetHandler) writeAccepted(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(passwordResetAcceptedResponse{
		Message: passwordResetAcceptedMessage,
	})
}
