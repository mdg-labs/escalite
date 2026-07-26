package handlers

import (
	"context"
	"encoding/xml"
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
	"github.com/mdg-labs/escalite/services/api/internal/crypto"
	"github.com/mdg-labs/escalite/services/api/internal/db"
	escalitesaml "github.com/mdg-labs/escalite/services/api/internal/saml"
)

// SAMLHandler handles optional SAML SSO login when configured for the organization.
type SAMLHandler struct {
	pool       *pgxpool.Pool
	logger     *slog.Logger
	secrets    *crypto.Box
	publicURL  string
	successURL string
	audit      *audit.Recorder
}

// NewSAMLHandler returns a handler for SAML login and ACS routes.
func NewSAMLHandler(pool *pgxpool.Pool, logger *slog.Logger, secrets *crypto.Box, publicURL, successURL string) *SAMLHandler {
	return &SAMLHandler{
		pool:       pool,
		logger:     logger,
		secrets:    secrets,
		publicURL:  publicURL,
		successURL: successURL,
		audit:      audit.NewRecorder(logger),
	}
}

// Login initiates SP-initiated SAML login when SAML is enabled.
func (h *SAMLHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteAPIError(w, http.StatusMethodNotAllowed, CodeMethodNotAllowed, "method not allowed")
		return
	}

	settings, err := h.loadEnabledSettings(r.Context())
	if err != nil {
		if errors.Is(err, errSAMLDisabled) {
			WriteAPIError(w, http.StatusNotFound, CodeNotFound, "saml login is not enabled")
			return
		}
		h.logger.Error("load saml settings failed", "error", err)
		WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
		return
	}

	sp, err := h.serviceProvider(settings)
	if err != nil {
		h.logger.Error("build saml provider failed", "error", err)
		WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
		return
	}

	relayState, err := escalitesaml.GenerateRelayState()
	if err != nil {
		h.logger.Error("generate saml relay state failed", "error", err)
		WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
		return
	}

	redirectURL, requestID, err := escalitesaml.LoginRedirectURL(sp, relayState)
	if err != nil {
		h.logger.Error("create saml login redirect failed", "error", err)
		WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
		return
	}

	escalitesaml.SetFlowCookies(w, requestID, relayState)
	http.Redirect(w, r, redirectURL.String(), http.StatusFound)
}

// ACS completes the SAML flow, validates the response, and establishes a session.
func (h *SAMLHandler) ACS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteAPIError(w, http.StatusMethodNotAllowed, CodeMethodNotAllowed, "method not allowed")
		return
	}

	settings, err := h.loadEnabledSettings(r.Context())
	if err != nil {
		if errors.Is(err, errSAMLDisabled) {
			WriteAPIError(w, http.StatusNotFound, CodeNotFound, "saml login is not enabled")
			return
		}
		h.logger.Error("load saml settings failed", "error", err)
		WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
		return
	}

	sp, err := h.serviceProvider(settings)
	if err != nil {
		h.logger.Error("build saml provider failed", "error", err)
		WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
		return
	}

	requestID, _ := escalitesaml.FlowCookies(r)
	assertion, err := escalitesaml.ParseACSResponse(sp, r, requestID)
	if err != nil {
		h.logger.Warn("saml response validation failed", "error", err)
		WriteAPIError(w, http.StatusUnauthorized, CodeUnauthenticated, "saml authentication failed")
		return
	}
	escalitesaml.ClearFlowCookies(w)

	email := strings.ToLower(strings.TrimSpace(escalitesaml.EmailFromAssertion(assertion)))
	if _, err := mail.ParseAddress(email); err != nil {
		h.logger.Warn("saml email claim invalid", "email", email)
		WriteAPIError(w, http.StatusUnauthorized, CodeUnauthenticated, "saml authentication failed")
		return
	}

	user, err := h.resolveUser(r.Context(), email)
	if err != nil {
		if errors.Is(err, errSAMLSetupRequired) {
			WriteAPIError(w, http.StatusForbidden, CodeForbidden, "setup is required before saml login")
			return
		}
		if errors.Is(err, errSAMLUserDeprovisioned) {
			WriteAPIError(w, http.StatusUnauthorized, CodeUnauthenticated, "saml authentication failed")
			return
		}
		h.logger.Error("resolve saml user failed", "error", err)
		WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
		return
	}

	if err := h.createSession(r.Context(), w, r, user); err != nil {
		h.logger.Error("create saml session failed", "error", err)
		WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
		return
	}

	http.Redirect(w, r, h.successURL, http.StatusFound)
}

// Metadata serves SP metadata for IdP configuration.
func (h *SAMLHandler) Metadata(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteAPIError(w, http.StatusMethodNotAllowed, CodeMethodNotAllowed, "method not allowed")
		return
	}

	settings, err := h.loadEnabledSettings(r.Context())
	if err != nil {
		if errors.Is(err, errSAMLDisabled) {
			WriteAPIError(w, http.StatusNotFound, CodeNotFound, "saml login is not enabled")
			return
		}
		h.logger.Error("load saml settings failed", "error", err)
		WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
		return
	}

	sp, err := h.serviceProvider(settings)
	if err != nil {
		h.logger.Error("build saml provider failed", "error", err)
		WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
		return
	}

	payload, err := xml.MarshalIndent(sp.Metadata(), "", "  ")
	if err != nil {
		h.logger.Error("marshal saml metadata failed", "error", err)
		WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
		return
	}

	w.Header().Set("Content-Type", "application/samlmetadata+xml")
	_, _ = w.Write(payload)
}

var (
	errSAMLDisabled           = errors.New("saml is disabled")
	errSAMLSetupRequired      = errors.New("organization setup required")
	errSAMLUserDeprovisioned  = errors.New("user is deprovisioned")
)

func (h *SAMLHandler) loadEnabledSettings(ctx context.Context) (db.OrganizationSamlSetting, error) {
	queries := db.New(h.pool)
	settings, err := queries.GetEnabledOrganizationSamlSettings(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.OrganizationSamlSetting{}, errSAMLDisabled
		}
		return db.OrganizationSamlSetting{}, err
	}
	if !settings.Enabled {
		return db.OrganizationSamlSetting{}, errSAMLDisabled
	}
	return settings, nil
}

func (h *SAMLHandler) serviceProvider(settings db.OrganizationSamlSetting) (*escalitesaml.ServiceProvider, error) {
	privateKeyPEM, err := h.secrets.Decrypt(crypto.Encrypted{
		KeyID:      settings.SpEncryptionKeyID,
		Ciphertext: settings.SpPrivateKeyCiphertext,
	})
	if err != nil {
		return nil, fmt.Errorf("decrypt sp private key: %w", err)
	}

	return escalitesaml.BuildServiceProvider(escalitesaml.ProviderConfig{
		PublicURL:         h.publicURL,
		IDPEntityID:       settings.IdpEntityID,
		IDPSSOURL:         settings.IdpSsoUrl,
		IDPCertificatePEM: settings.IdpCertificatePem,
		SPCertificatePEM:  settings.SpCertificatePem,
		SPPrivateKeyPEM:   string(privateKeyPEM),
	})
}

func (h *SAMLHandler) resolveUser(ctx context.Context, email string) (db.User, error) {
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
			return db.User{}, errSAMLSetupRequired
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

func (h *SAMLHandler) createSession(ctx context.Context, w http.ResponseWriter, r *http.Request, user db.User) error {
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
