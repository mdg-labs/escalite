package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/mail"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mdg-labs/escalite/services/api/internal/alerts"
	"github.com/mdg-labs/escalite/services/api/internal/auth"
	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/engine/escalationapi"
	"github.com/mdg-labs/escalite/services/integrations"
)

const (
	emailToAlertPluginName              = "email-to-alert"
	inboundEmailRelaySecretHeader       = "X-Escalite-Relay-Secret"
	inboundEmailAuthenticatedHeader     = "X-Escalite-Email-Authenticated"
	inboundEmailAuthenticatedPass       = "pass"
	inboundEmailMaxBodyBytes            = 256 * 1024
)

// InboundEmailConfig controls inbound email ingestion from a trusted SMTP relay.
type InboundEmailConfig struct {
	RelaySecret          string
	Domain               string
	RequireAuthenticated bool
}

// InboundEmailHandler handles POST /api/v1/inbound/email from a trusted SMTP relay.
type InboundEmailHandler struct {
	pool   *pgxpool.Pool
	jobs   escalationapi.JobProducer
	logger *slog.Logger
	cfg    InboundEmailConfig
}

// NewInboundEmailHandler returns a handler for inbound email relay requests.
func NewInboundEmailHandler(pool *pgxpool.Pool, jobs escalationapi.JobProducer, logger *slog.Logger, cfg InboundEmailConfig) *InboundEmailHandler {
	return &InboundEmailHandler{
		pool:   pool,
		jobs:   jobs,
		logger: logger,
		cfg:    cfg,
	}
}

type inboundEmailRelayPayload struct {
	To            string `json:"to"`
	From          string `json:"from"`
	Subject       string `json:"subject"`
	Text          string `json:"text"`
	HTML          string `json:"html"`
	MessageID     string `json:"message_id"`
	Authenticated bool   `json:"authenticated"`
}

type inboundEmailResponse struct {
	Status string `json:"status"`
}

// Enabled reports whether inbound email ingestion is configured.
func (cfg InboundEmailConfig) Enabled() bool {
	return strings.TrimSpace(cfg.RelaySecret) != ""
}

// ServeHTTP validates relay authentication and creates alerts from parsed email payloads.
func (h *InboundEmailHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteAPIError(w, http.StatusMethodNotAllowed, CodeMethodNotAllowed, "method not allowed")
		return
	}

	if !h.cfg.Enabled() {
		WriteAPIError(w, http.StatusServiceUnavailable, CodeInternal, "inbound email is not configured")
		return
	}

	if !h.authenticatedRelay(r) {
		WriteAPIError(w, http.StatusUnauthorized, CodeUnauthenticated, "relay authentication required")
		return
	}

	if !isJSONContentType(r.Header.Get("Content-Type")) {
		WriteAPIError(w, http.StatusBadRequest, CodeValidation, "content-type must be application/json")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, inboundEmailMaxBodyBytes)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			WriteAPIError(w, http.StatusRequestEntityTooLarge, CodePayloadTooLarge, "request body too large")
			return
		}
		h.logger.Error("read inbound email body failed", "error", err)
		WriteAPIError(w, http.StatusBadRequest, CodeValidation, "invalid request body")
		return
	}

	var relay inboundEmailRelayPayload
	if err := json.Unmarshal(body, &relay); err != nil {
		WriteAPIError(w, http.StatusBadRequest, CodeValidation, "invalid email payload")
		return
	}

	if !h.emailAuthenticated(r, relay) {
		WriteAPIError(w, http.StatusForbidden, CodeForbidden, "unsigned or unauthenticated email rejected")
		return
	}

	token, err := integrationKeyTokenFromRecipient(relay.To, h.cfg.Domain)
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, CodeValidation, "invalid recipient address")
		return
	}

	tokenHash := auth.HashPasswordResetToken(token)
	ctx := r.Context()
	queries := db.New(h.pool)

	key, err := queries.GetActiveIntegrationKeyByTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteAPIError(w, http.StatusNotFound, CodeNotFound, "not found")
			return
		}
		h.logger.Error("lookup integration key failed", "error", err)
		WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
		return
	}

	if key.PluginName != emailToAlertPluginName {
		WriteAPIError(w, http.StatusNotFound, CodeNotFound, "not found")
		return
	}

	plugin, err := integrations.Get(emailToAlertPluginName)
	if err != nil {
		h.logger.Error("email-to-alert plugin missing", "error", err)
		WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
		return
	}

	parsedAlerts, err := integrations.ParseAll(plugin, body, r.Header, key.Config)
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, CodeValidation, "invalid email payload")
		return
	}

	for _, alertEvent := range parsedAlerts {
		if _, err := alerts.ProcessInbound(ctx, queries, h.logger, key, alertEvent, &alerts.InboundDeps{
			Pool: h.pool,
			Jobs: h.jobs,
		}); err != nil {
			h.logger.Error("process inbound email alert failed",
				"integration_key_id", key.ID,
				"service_id", key.ServiceID,
				"plugin", emailToAlertPluginName,
				"dedup_key", alertEvent.DedupKey,
				"error", err,
			)
			WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
			return
		}
	}

	h.logger.Info("inbound email accepted",
		"integration_key_id", key.ID,
		"service_id", key.ServiceID,
		"token_prefix", key.Prefix,
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(inboundEmailResponse{Status: "accepted"})
}

func (h *InboundEmailHandler) authenticatedRelay(r *http.Request) bool {
	secret := strings.TrimSpace(h.cfg.RelaySecret)
	if secret == "" {
		return false
	}

	if headerSecret := strings.TrimSpace(r.Header.Get(inboundEmailRelaySecretHeader)); headerSecret != "" {
		return headerSecret == secret
	}

	if bearer, ok := auth.ParseBearerToken(r.Header.Get("Authorization")); ok {
		return bearer == secret
	}

	return false
}

func (h *InboundEmailHandler) emailAuthenticated(r *http.Request, relay inboundEmailRelayPayload) bool {
	if !h.cfg.RequireAuthenticated {
		return true
	}
	if relay.Authenticated {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(r.Header.Get(inboundEmailAuthenticatedHeader)), inboundEmailAuthenticatedPass)
}

func integrationKeyTokenFromRecipient(recipient, domain string) (string, error) {
	recipient = strings.TrimSpace(recipient)
	if recipient == "" {
		return "", errors.New("recipient is required")
	}

	addr, err := mail.ParseAddress(recipient)
	if err != nil {
		return "", err
	}

	local, host, ok := strings.Cut(addr.Address, "@")
	if !ok || local == "" || host == "" {
		return "", errors.New("invalid recipient address")
	}

	domain = strings.ToLower(strings.TrimSpace(domain))
	if domain != "" && strings.ToLower(host) != domain {
		return "", errors.New("recipient domain mismatch")
	}

	return local, nil
}

// InboundEmailAddress returns the unique inbound address for an integration key token.
func InboundEmailAddress(token, domain string) string {
	token = strings.TrimSpace(token)
	domain = strings.TrimSpace(domain)
	if token == "" || domain == "" {
		return ""
	}
	return token + "@" + strings.TrimPrefix(domain, "@")
}
