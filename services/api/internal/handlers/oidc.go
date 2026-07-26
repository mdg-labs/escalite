package handlers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mdg-labs/escalite/services/api/internal/audit"
	"github.com/mdg-labs/escalite/services/api/internal/auth"
	"github.com/mdg-labs/escalite/services/api/internal/authz"
	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/api/internal/oidc"
)

// OIDCAuthenticator exchanges an authorization code for verified user identity.
type OIDCAuthenticator interface {
	AuthCodeURL(state string) string
	Exchange(ctx context.Context, code string) (oidc.UserInfo, error)
}

// OIDCHandler handles optional OIDC login when configured.
type OIDCHandler struct {
	pool        *pgxpool.Pool
	logger      *slog.Logger
	provider    OIDCAuthenticator
	successURL  string
	audit       *audit.Recorder
}

// NewOIDCHandler returns a handler for OIDC login and callback routes.
func NewOIDCHandler(pool *pgxpool.Pool, logger *slog.Logger, provider OIDCAuthenticator, successURL string) *OIDCHandler {
	return &OIDCHandler{
		pool:       pool,
		logger:     logger,
		provider:   provider,
		successURL: successURL,
		audit:      audit.NewRecorder(logger),
	}
}

// Login initiates the OIDC authorization code flow.
func (h *OIDCHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteAPIError(w, http.StatusMethodNotAllowed, CodeMethodNotAllowed, "method not allowed")
		return
	}

	state, err := oidc.GenerateState()
	if err != nil {
		h.logger.Error("generate oidc state failed", "error", err)
		WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
		return
	}

	oidc.SetStateCookie(w, state)
	http.Redirect(w, r, h.provider.AuthCodeURL(state), http.StatusFound)
}

// Callback completes the OIDC flow, creates or links the user, and establishes a session.
func (h *OIDCHandler) Callback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteAPIError(w, http.StatusMethodNotAllowed, CodeMethodNotAllowed, "method not allowed")
		return
	}

	if errParam := strings.TrimSpace(r.URL.Query().Get("error")); errParam != "" {
		h.logger.Warn("oidc provider returned error", "error", errParam)
		WriteAPIError(w, http.StatusUnauthorized, CodeUnauthenticated, "oidc authentication failed")
		return
	}

	state := strings.TrimSpace(r.URL.Query().Get("state"))
	if !oidc.ValidateStateCookie(r, state) {
		WriteAPIError(w, http.StatusUnauthorized, CodeUnauthenticated, "invalid oidc state")
		return
	}
	oidc.ClearStateCookie(w)

	code := strings.TrimSpace(r.URL.Query().Get("code"))
	if code == "" {
		WriteAPIError(w, http.StatusBadRequest, CodeValidation, "authorization code is required")
		return
	}

	ctx := r.Context()
	userInfo, err := h.provider.Exchange(ctx, code)
	if err != nil {
		h.logger.Error("oidc exchange failed", "error", err)
		WriteAPIError(w, http.StatusUnauthorized, CodeUnauthenticated, "oidc authentication failed")
		return
	}

	if _, err := mail.ParseAddress(userInfo.Email); err != nil {
		h.logger.Warn("oidc email claim invalid", "email", userInfo.Email)
		WriteAPIError(w, http.StatusUnauthorized, CodeUnauthenticated, "oidc authentication failed")
		return
	}

	user, err := h.resolveUser(ctx, userInfo.Email)
	if err != nil {
		if errors.Is(err, errOIDCSetupRequired) {
			WriteAPIError(w, http.StatusForbidden, CodeForbidden, "setup is required before oidc login")
			return
		}
		if errors.Is(err, errOIDCUserDeprovisioned) {
			WriteAPIError(w, http.StatusUnauthorized, CodeUnauthenticated, "oidc authentication failed")
			return
		}
		h.logger.Error("resolve oidc user failed", "error", err)
		WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
		return
	}

	if err := h.createSession(ctx, w, r, user); err != nil {
		h.logger.Error("create oidc session failed", "error", err)
		WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
		return
	}

	http.Redirect(w, r, h.successURL, http.StatusFound)
}

var errOIDCSetupRequired = errors.New("organization setup required")
var errOIDCUserDeprovisioned = errors.New("user is deprovisioned")

func (h *OIDCHandler) resolveUser(ctx context.Context, email string) (db.User, error) {
	queries := db.New(h.pool)

	account, accountErr := queries.GetAccountByEmail(ctx, email)
	if accountErr == nil {
		user, err := auth.LoginMembership(ctx, queries, account.ID)
		if err == nil {
			return user, nil
		}
		if !errors.Is(err, auth.ErrNoActiveMembership) {
			return db.User{}, err
		}
	} else if !errors.Is(accountErr, pgx.ErrNoRows) {
		return db.User{}, accountErr
	}

	org, err := queries.GetFirstOrganization(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.User{}, errOIDCSetupRequired
		}
		return db.User{}, err
	}

	if accountErr == nil {
		return auth.EnsureOrgMembership(ctx, queries, account, org.ID, authz.RoleMember)
	}

	account, err = auth.FindOrCreateAccountByEmail(ctx, queries, email)
	if err != nil {
		return db.User{}, err
	}

	return auth.EnsureOrgMembership(ctx, queries, account, org.ID, authz.RoleMember)
}

func (h *OIDCHandler) createSession(ctx context.Context, w http.ResponseWriter, r *http.Request, user db.User) error {
	queries := db.New(h.pool)

	sessionID := uuid.Must(uuid.NewV7())
	expiresAt := time.Now().UTC().Add(auth.DefaultSessionTTL)
	userAgent := r.UserAgent()

	_, err := queries.CreateSession(ctx, db.CreateSessionParams{
		ID:             sessionID,
		UserID:         user.ID,
		OrganizationID: user.OrganizationID,
		ExpiresAt:      pgtype.Timestamptz{Time: expiresAt, Valid: true},
		UserAgent:      pgtype.Text{String: userAgent, Valid: userAgent != ""},
	})
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	auth.SetSessionCookie(w, sessionID, expiresAt)
	h.audit.Login(ctx, queries, user.OrganizationID, user.ID, audit.RequestMetaFromHTTP(r))
	return nil
}
