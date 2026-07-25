package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mdg-labs/escalite/services/api/internal/alerts"
	"github.com/mdg-labs/escalite/services/api/internal/auth"
	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/engine/escalationapi"
	"github.com/mdg-labs/escalite/services/integrations"
)

const genericRESTPluginName = "generic-rest-api"

// InboundAlertsHandler handles POST /api/v1/alerts.
type InboundAlertsHandler struct {
	pool   *pgxpool.Pool
	jobs   escalationapi.JobProducer
	logger *slog.Logger
}

// NewInboundAlertsHandler returns a handler for authenticated alert creation.
func NewInboundAlertsHandler(pool *pgxpool.Pool, jobs escalationapi.JobProducer, logger *slog.Logger) *InboundAlertsHandler {
	return &InboundAlertsHandler{
		pool:   pool,
		jobs:   jobs,
		logger: logger,
	}
}

type inboundAlertResponse struct {
	ID string `json:"id"`
}

// ServeHTTP creates an alert from a generic-rest-api integration key bearer token.
func (h *InboundAlertsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteAPIError(w, http.StatusMethodNotAllowed, CodeMethodNotAllowed, "method not allowed")
		return
	}

	token, ok := auth.ParseBearerToken(r.Header.Get("Authorization"))
	if !ok {
		WriteAPIError(w, http.StatusUnauthorized, CodeUnauthenticated, "authentication required")
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
		h.logger.Error("read alert body failed", "error", err)
		WriteAPIError(w, http.StatusBadRequest, CodeValidation, "invalid request body")
		return
	}

	tokenHash := auth.HashPasswordResetToken(token)
	ctx := r.Context()
	queries := db.New(h.pool)

	key, err := queries.GetActiveIntegrationKeyByTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteAPIError(w, http.StatusUnauthorized, CodeUnauthenticated, "authentication required")
			return
		}
		h.logger.Error("lookup integration key failed", "error", err)
		WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
		return
	}

	if key.PluginName != genericRESTPluginName {
		WriteAPIError(w, http.StatusUnauthorized, CodeUnauthenticated, "authentication required")
		return
	}

	plugin, err := integrations.Get(genericRESTPluginName)
	if err != nil {
		h.logger.Error("generic-rest-api plugin missing", "error", err)
		WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
		return
	}

	alertEvent, err := plugin.ParseAlert(body, r.Header)
	if err != nil {
		WriteAPIError(w, http.StatusBadRequest, CodeValidation, "invalid alert payload")
		return
	}

	alertID, err := alerts.ProcessInbound(ctx, queries, h.logger, key, alertEvent, &alerts.InboundDeps{
		Pool: h.pool,
		Jobs: h.jobs,
	})
	if err != nil {
		h.logger.Error("process inbound alert failed",
			"integration_key_id", key.ID,
			"service_id", key.ServiceID,
			"plugin", genericRESTPluginName,
			"dedup_key", alertEvent.DedupKey,
			"error", err,
		)
		WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
		return
	}

	h.logger.Info("alert created via generic-rest-api",
		"integration_key_id", key.ID,
		"service_id", key.ServiceID,
		"token_prefix", key.Prefix,
		"alert_id", alertID,
	)

	w.Header().Set("Content-Type", "application/json")
	if alertID == uuid.Nil {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "resolved"})
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(inboundAlertResponse{ID: alertID.String()})
}
