package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mdg-labs/escalite/services/api/internal/auth"
	"github.com/mdg-labs/escalite/services/api/internal/alerts"
	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/api/internal/ratelimit"
	"github.com/mdg-labs/escalite/services/integrations"
)

const (
	webhookMaxBodyBytes       = 256 * 1024
	defaultWebhookRateLimit   = 120
	defaultWebhookRateWindow  = time.Minute
	CodePayloadTooLarge       = "PAYLOAD_TOO_LARGE"
)

// InboundWebhookConfig controls inbound webhook ingestion behavior.
type InboundWebhookConfig struct {
	KeyLimiter ratelimit.Limiter
}

// InboundWebhookHandler handles POST /webhook/{plugin}/{token}.
type InboundWebhookHandler struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
	cfg    InboundWebhookConfig
}

// NewInboundWebhookHandler returns a handler for inbound webhook requests.
func NewInboundWebhookHandler(pool *pgxpool.Pool, logger *slog.Logger, cfg InboundWebhookConfig) *InboundWebhookHandler {
	if cfg.KeyLimiter == nil {
		cfg.KeyLimiter = ratelimit.NewMemoryLimiter(defaultWebhookRateLimit, defaultWebhookRateWindow)
	}
	return &InboundWebhookHandler{
		pool:   pool,
		logger: logger,
		cfg:    cfg,
	}
}

type inboundWebhookResponse struct {
	Status string `json:"status"`
}

// ServeHTTP validates and accepts an inbound webhook payload.
func (h *InboundWebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteAPIError(w, http.StatusMethodNotAllowed, CodeMethodNotAllowed, "method not allowed")
		return
	}

	pluginName := strings.TrimSpace(chi.URLParam(r, "plugin"))
	token := strings.TrimSpace(chi.URLParam(r, "token"))
	if pluginName == "" || token == "" {
		WriteAPIError(w, http.StatusNotFound, CodeNotFound, "not found")
		return
	}

	if !isJSONContentType(r.Header.Get("Content-Type")) {
		WriteAPIError(w, http.StatusBadRequest, CodeValidation, "content-type must be application/json")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, webhookMaxBodyBytes)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			WriteAPIError(w, http.StatusRequestEntityTooLarge, CodePayloadTooLarge, "request body too large")
			return
		}
		h.logger.Error("read webhook body failed", "error", err)
		WriteAPIError(w, http.StatusBadRequest, CodeValidation, "invalid request body")
		return
	}

	if _, err := integrations.Get(pluginName); err != nil {
		WriteAPIError(w, http.StatusNotFound, CodeNotFound, "not found")
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

	if key.PluginName != pluginName {
		WriteAPIError(w, http.StatusNotFound, CodeNotFound, "not found")
		return
	}

	if !h.cfg.KeyLimiter.Allow("webhook:" + key.ID.String()) {
		WriteAPIError(w, http.StatusTooManyRequests, CodeRateLimited, "too many webhook requests for this integration key")
		return
	}

	plugin, err := integrations.Get(pluginName)
	if err != nil {
		WriteAPIError(w, http.StatusNotFound, CodeNotFound, "not found")
		return
	}

	parsedAlerts, err := integrations.ParseAll(plugin, body, r.Header, key.Config)
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, CodeValidation, "invalid webhook payload")
		return
	}

	for _, alertEvent := range parsedAlerts {
		if _, err := alerts.ProcessInbound(ctx, queries, h.logger, key, alertEvent); err != nil {
			h.logger.Error("process inbound alert failed",
				"integration_key_id", key.ID,
				"service_id", key.ServiceID,
				"plugin", pluginName,
				"dedup_key", alertEvent.DedupKey,
				"error", err,
			)
			WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
			return
		}
	}

	h.logger.Info("webhook received",
		"integration_key_id", key.ID,
		"service_id", key.ServiceID,
		"plugin", pluginName,
		"token_prefix", key.Prefix,
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(inboundWebhookResponse{Status: "accepted"})
}

func isJSONContentType(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return false
	}
	return strings.EqualFold(mediaType, "application/json")
}
