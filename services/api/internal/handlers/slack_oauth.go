package handlers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mdg-labs/escalite/services/api/internal/audit"
	"github.com/mdg-labs/escalite/services/api/internal/auth"
	"github.com/mdg-labs/escalite/services/api/internal/authz"
	"github.com/mdg-labs/escalite/services/api/internal/crypto"
	"github.com/mdg-labs/escalite/services/api/internal/db"
)

const (
	slackOAuthStateCookieName = "escalite_slack_oauth_state"
	slackOAuthCookiePath      = "/api/v1/integrations/slack"
	slackOAuthStateTTL        = 10 * time.Minute

	slackOAuthAuthorizeURL = "https://slack.com/oauth/v2/authorize"
	slackOAuthExchangeURL  = "https://slack.com/api/oauth.v2.access"
)

// SlackInstallResult is the outcome of a successful Slack OAuth v2 bot token exchange.
type SlackInstallResult struct {
	BotToken      string
	WorkspaceID   string
	WorkspaceName string
	BotUserID     string
	Scope         string
}

// SlackOAuthProvider builds the Slack "Add to Slack" authorize URL and exchanges an
// authorization code for a workspace bot token.
type SlackOAuthProvider interface {
	AuthCodeURL(state string) string
	Exchange(ctx context.Context, code string) (SlackInstallResult, error)
}

// SlackOAuthHandler handles the Slack app install/reinstall OAuth flow (org admin only).
// Reinstalling always upserts the single row for the admin's organization (see
// queries/organization_slack_settings.sql UpsertOrganizationSlackOAuthInstall), so
// re-authorizing never creates a duplicate workspace row.
type SlackOAuthHandler struct {
	pool       *pgxpool.Pool
	logger     *slog.Logger
	provider   SlackOAuthProvider
	secrets    *crypto.Box
	successURL string
	audit      *audit.Recorder
}

// NewSlackOAuthHandler returns a handler for the Slack app install/callback routes.
func NewSlackOAuthHandler(pool *pgxpool.Pool, logger *slog.Logger, provider SlackOAuthProvider, secrets *crypto.Box, successURL string) *SlackOAuthHandler {
	return &SlackOAuthHandler{
		pool:       pool,
		logger:     logger,
		provider:   provider,
		secrets:    secrets,
		successURL: successURL,
		audit:      audit.NewRecorder(logger),
	}
}

// Install starts the Slack OAuth authorization code flow for the authenticated org admin.
// Must run behind session middleware (see server.New's session-protected route group).
func (h *SlackOAuthHandler) Install(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteAPIError(w, http.StatusMethodNotAllowed, CodeMethodNotAllowed, "method not allowed")
		return
	}

	if _, err := h.requireAdmin(r); err != nil {
		writeSlackOAuthAuthError(w, err)
		return
	}

	state, err := generateSlackOAuthState()
	if err != nil {
		h.logger.Error("generate slack oauth state failed", "error", err)
		WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
		return
	}

	setSlackOAuthStateCookie(w, state)
	http.Redirect(w, r, h.provider.AuthCodeURL(state), http.StatusFound)
}

// Callback completes the Slack OAuth flow and upserts the organization's workspace install.
func (h *SlackOAuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteAPIError(w, http.StatusMethodNotAllowed, CodeMethodNotAllowed, "method not allowed")
		return
	}

	if errParam := strings.TrimSpace(r.URL.Query().Get("error")); errParam != "" {
		h.logger.Warn("slack oauth provider returned error", "error", errParam)
		clearSlackOAuthStateCookie(w)
		h.redirectWithStatus(w, r, "error")
		return
	}

	state := strings.TrimSpace(r.URL.Query().Get("state"))
	if !validateSlackOAuthStateCookie(r, state) {
		WriteAPIError(w, http.StatusUnauthorized, CodeUnauthenticated, "invalid oauth state")
		return
	}
	clearSlackOAuthStateCookie(w)

	sc, err := h.requireAdmin(r)
	if err != nil {
		writeSlackOAuthAuthError(w, err)
		return
	}

	code := strings.TrimSpace(r.URL.Query().Get("code"))
	if code == "" {
		WriteAPIError(w, http.StatusBadRequest, CodeValidation, "authorization code is required")
		return
	}

	ctx := r.Context()
	result, err := h.provider.Exchange(ctx, code)
	if err != nil {
		h.logger.Error("slack oauth exchange failed", "error", err)
		h.redirectWithStatus(w, r, "error")
		return
	}

	if err := h.storeInstall(ctx, sc, result); err != nil {
		h.logger.Error("store slack oauth install failed", "error", err)
		h.redirectWithStatus(w, r, "error")
		return
	}

	h.redirectWithStatus(w, r, "connected")
}

var errSlackOAuthUnauthenticated = errors.New("authentication required")
var errSlackOAuthForbidden = errors.New("admin access required")

func (h *SlackOAuthHandler) requireAdmin(r *http.Request) (auth.SessionContext, error) {
	sc, ok := auth.SessionFromContext(r.Context())
	if !ok {
		return auth.SessionContext{}, errSlackOAuthUnauthenticated
	}
	if !authz.IsAdmin(sc.User.Role) {
		return auth.SessionContext{}, errSlackOAuthForbidden
	}
	return sc, nil
}

func writeSlackOAuthAuthError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errSlackOAuthForbidden):
		WriteAPIError(w, http.StatusForbidden, CodeForbidden, "admin access required")
	default:
		WriteAPIError(w, http.StatusUnauthorized, CodeUnauthenticated, "authentication required")
	}
}

func (h *SlackOAuthHandler) storeInstall(ctx context.Context, sc auth.SessionContext, result SlackInstallResult) error {
	if h.secrets == nil {
		return errors.New("encryption is not configured")
	}

	botToken := strings.TrimSpace(result.BotToken)
	if botToken == "" {
		return errors.New("slack oauth response missing bot token")
	}

	encrypted, err := h.secrets.Encrypt([]byte(botToken))
	if err != nil {
		return fmt.Errorf("encrypt slack bot token: %w", err)
	}

	queries := db.New(h.pool)
	if _, err := queries.UpsertOrganizationSlackOAuthInstall(ctx, db.UpsertOrganizationSlackOAuthInstallParams{
		OrganizationID:     sc.User.OrganizationID,
		BotTokenCiphertext: encrypted.Ciphertext,
		EncryptionKeyID:    encrypted.KeyID,
		TokenHint:          crypto.SecretHint(botToken),
		WorkspaceID:        textOrInvalid(result.WorkspaceID),
		WorkspaceName:      textOrInvalid(result.WorkspaceName),
		BotUserID:          textOrInvalid(result.BotUserID),
		Scope:              textOrInvalid(result.Scope),
	}); err != nil {
		return fmt.Errorf("upsert slack oauth install: %w", err)
	}

	h.audit.SlackWorkspaceConnected(ctx, db.New(h.pool), sc.User.OrganizationID, sc.User.ID, result.WorkspaceID)
	return nil
}

func textOrInvalid(value string) pgtype.Text {
	trimmed := strings.TrimSpace(value)
	return pgtype.Text{String: trimmed, Valid: trimmed != ""}
}

func (h *SlackOAuthHandler) redirectWithStatus(w http.ResponseWriter, r *http.Request, status string) {
	target := h.successURL
	if target == "" {
		target = "/"
	}
	separator := "?"
	if strings.Contains(target, "?") {
		separator = "&"
	}
	http.Redirect(w, r, target+separator+"slack="+status, http.StatusFound)
}

func generateSlackOAuthState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate slack oauth state: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func setSlackOAuthStateCookie(w http.ResponseWriter, state string) {
	http.SetCookie(w, &http.Cookie{
		Name:     slackOAuthStateCookieName,
		Value:    state,
		Path:     slackOAuthCookiePath,
		MaxAge:   int(slackOAuthStateTTL.Seconds()),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}

func validateSlackOAuthStateCookie(r *http.Request, callbackState string) bool {
	cookie, err := r.Cookie(slackOAuthStateCookieName)
	if err != nil || cookie.Value == "" || callbackState == "" {
		return false
	}
	return cookie.Value == callbackState
}

func clearSlackOAuthStateCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     slackOAuthStateCookieName,
		Value:    "",
		Path:     slackOAuthCookiePath,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}

// slackOAuthClient is the production Slack OAuth v2 client for bot token installs.
type slackOAuthClient struct {
	clientID     string
	clientSecret string
	redirectURL  string
	scopes       []string
	httpClient   *http.Client
}

// NewSlackOAuthClient returns a SlackOAuthProvider backed by Slack's real OAuth v2 endpoints.
func NewSlackOAuthClient(clientID, clientSecret, redirectURL string, scopes []string) SlackOAuthProvider {
	return &slackOAuthClient{
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURL:  redirectURL,
		scopes:       scopes,
		httpClient:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *slackOAuthClient) AuthCodeURL(state string) string {
	q := url.Values{}
	q.Set("client_id", c.clientID)
	q.Set("scope", strings.Join(c.scopes, ","))
	q.Set("redirect_uri", c.redirectURL)
	q.Set("state", state)
	return slackOAuthAuthorizeURL + "?" + q.Encode()
}

func (c *slackOAuthClient) Exchange(ctx context.Context, code string) (SlackInstallResult, error) {
	form := url.Values{}
	form.Set("client_id", c.clientID)
	form.Set("client_secret", c.clientSecret)
	form.Set("code", code)
	form.Set("redirect_uri", c.redirectURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, slackOAuthExchangeURL, strings.NewReader(form.Encode()))
	if err != nil {
		return SlackInstallResult{}, fmt.Errorf("create slack oauth exchange request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return SlackInstallResult{}, fmt.Errorf("call slack oauth exchange: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return SlackInstallResult{}, fmt.Errorf("read slack oauth exchange response: %w", err)
	}

	var payload struct {
		OK          bool   `json:"ok"`
		Error       string `json:"error"`
		AccessToken string `json:"access_token"`
		Scope       string `json:"scope"`
		BotUserID   string `json:"bot_user_id"`
		Team        struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"team"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return SlackInstallResult{}, fmt.Errorf("parse slack oauth exchange response: %w", err)
	}
	if !payload.OK {
		if payload.Error == "" {
			payload.Error = "slack oauth exchange returned ok=false"
		}
		return SlackInstallResult{}, fmt.Errorf("slack oauth error: %s", payload.Error)
	}
	if strings.TrimSpace(payload.AccessToken) == "" {
		return SlackInstallResult{}, errors.New("slack oauth response missing access_token")
	}

	return SlackInstallResult{
		BotToken:      payload.AccessToken,
		WorkspaceID:   payload.Team.ID,
		WorkspaceName: payload.Team.Name,
		BotUserID:     payload.BotUserID,
		Scope:         payload.Scope,
	}, nil
}
