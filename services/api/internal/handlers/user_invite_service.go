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
)

const defaultInviteTokenTTL = 7 * 24 * time.Hour

// UserInviteService provisions organization users and sends invite emails.
type UserInviteService struct {
	pool    *pgxpool.Pool
	queries db.Querier
	mail    email.Sender
	cfg     UserInviteConfig
	logger  interface {
		Error(msg string, args ...any)
	}
}

// UserInviteConfig controls invite email behavior.
type UserInviteConfig struct {
	PublicURL    string
	InviteTokenTTL time.Duration
}

// NewUserInviteService returns a user invite service.
func NewUserInviteService(
	pool *pgxpool.Pool,
	queries db.Querier,
	mail email.Sender,
	cfg UserInviteConfig,
	logger interface {
		Error(msg string, args ...any)
	},
) *UserInviteService {
	if cfg.InviteTokenTTL <= 0 {
		cfg.InviteTokenTTL = defaultInviteTokenTTL
	}
	return &UserInviteService{
		pool:    pool,
		queries: queries,
		mail:    mail,
		cfg:     cfg,
		logger:  logger,
	}
}

// InviteUser creates or reactivates an organization user and optionally emails an invite.
func (s *UserInviteService) InviteUser(
	ctx context.Context,
	orgID, actorUserID uuid.UUID,
	orgName, emailAddr, role string,
) (db.User, error) {
	emailAddr = strings.ToLower(strings.TrimSpace(emailAddr))
	if emailAddr == "" {
		return db.User{}, &UserInviteError{Code: CodeValidation, Message: "email is required"}
	}
	if _, err := mail.ParseAddress(emailAddr); err != nil {
		return db.User{}, &UserInviteError{Code: CodeValidation, Message: "email is invalid"}
	}
	if role != "admin" && role != "member" {
		return db.User{}, &UserInviteError{Code: CodeValidation, Message: "role is invalid"}
	}

	actor, err := s.queries.GetUserByID(ctx, db.GetUserByIDParams{
		ID:             actorUserID,
		OrganizationID: orgID,
	})
	if err != nil {
		s.logger.Error("invite actor lookup failed", "error", err)
		return db.User{}, &UserInviteError{Code: CodeInternal, Message: "internal error"}
	}
	if strings.EqualFold(strings.TrimSpace(actor.Email), emailAddr) {
		return db.User{}, &UserInviteError{Code: CodeValidation, Message: "cannot invite yourself"}
	}

	existing, err := s.queries.GetUserByEmail(ctx, db.GetUserByEmailParams{
		OrganizationID: orgID,
		Email:          emailAddr,
	})
	if err == nil {
		if auth.IsUserActive(existing.DeprovisionedAt.Valid) {
			return db.User{}, &UserInviteError{Code: CodeValidation, Message: "user already exists"}
		}
		user, err := s.queries.ReprovisionUserByInvite(ctx, db.ReprovisionUserByInviteParams{
			ID:             existing.ID,
			OrganizationID: orgID,
			Role:           role,
		})
		if err != nil {
			s.logger.Error("reprovision invited user failed", "error", err)
			return db.User{}, &UserInviteError{Code: CodeInternal, Message: "internal error"}
		}
		if err := s.sendInviteEmail(ctx, user, orgName, emailAddr); err != nil {
			s.logger.Error("send invite email failed", "error", err)
			return db.User{}, &UserInviteError{Code: CodeInternal, Message: "internal error"}
		}
		return user, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		s.logger.Error("invite user lookup failed", "error", err)
		return db.User{}, &UserInviteError{Code: CodeInternal, Message: "internal error"}
	}

	account, err := auth.FindOrCreateAccountByEmail(ctx, s.queries, emailAddr)
	if err != nil {
		s.logger.Error("invite account lookup failed", "error", err)
		return db.User{}, &UserInviteError{Code: CodeInternal, Message: "internal error"}
	}

	user, err := auth.EnsureOrgMembership(ctx, s.queries, account, orgID, role)
	if err != nil {
		if errors.Is(err, auth.ErrNoActiveMembership) {
			return db.User{}, &UserInviteError{Code: CodeValidation, Message: "user already exists"}
		}
		s.logger.Error("invite membership create failed", "error", err)
		return db.User{}, &UserInviteError{Code: CodeInternal, Message: "internal error"}
	}

	if err := s.sendInviteEmail(ctx, user, orgName, emailAddr); err != nil {
		s.logger.Error("send invite email failed", "error", err)
		return db.User{}, &UserInviteError{Code: CodeInternal, Message: "internal error"}
	}

	return user, nil
}

// UserInviteError is a coded invite failure.
type UserInviteError struct {
	Code    string
	Message string
}

func (e *UserInviteError) Error() string {
	return e.Message
}

func (s *UserInviteService) sendInviteEmail(ctx context.Context, user db.User, orgName, emailAddr string) error {
	account, err := s.queries.GetAccountByID(ctx, user.AccountID)
	if err != nil {
		return err
	}

	hasPassword := account.PasswordHash.Valid && account.PasswordHash.String != ""
	if hasPassword {
		subject := "You've been added to " + orgName + " on Escalite"
		body := "An administrator added you to " + orgName + " on Escalite.\n\n" +
			"Sign in with your existing Escalite credentials to access the organization.\n"
		return s.mail.Send(ctx, email.Message{
			To:      emailAddr,
			Subject: subject,
			Body:    body,
		})
	}

	plaintext, tokenHash, err := auth.NewPasswordResetToken()
	if err != nil {
		return err
	}

	if err := s.queries.InvalidateUnusedPasswordResetTokensForUser(ctx, user.ID); err != nil {
		return err
	}

	expiresAt := time.Now().UTC().Add(s.cfg.InviteTokenTTL)
	if _, err := s.queries.CreatePasswordResetToken(ctx, db.CreatePasswordResetTokenParams{
		ID:             uuid.Must(uuid.NewV7()),
		UserID:         user.ID,
		OrganizationID: user.OrganizationID,
		TokenHash:      tokenHash,
		ExpiresAt:      pgtype.Timestamptz{Time: expiresAt, Valid: true},
	}); err != nil {
		return err
	}

	setupURL := buildPasswordResetURL(s.cfg.PublicURL, plaintext)
	subject := "You're invited to " + orgName + " on Escalite"
	body := "An administrator invited you to join " + orgName + " on Escalite.\n\n" +
		"Use the link below to set your password and sign in. The link expires in seven days.\n\n" + setupURL + "\n"
	if setupURL == plaintext {
		body = "An administrator invited you to join " + orgName + " on Escalite.\n\n" +
			"Use this token to set your password. The token expires in seven days.\n\n" + plaintext + "\n"
	}

	return s.mail.Send(ctx, email.Message{
		To:      emailAddr,
		Subject: subject,
		Body:    body,
	})
}
