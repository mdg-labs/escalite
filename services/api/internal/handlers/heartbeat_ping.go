package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mdg-labs/escalite/services/api/internal/auth"
	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/api/internal/ratelimit"
)

// HeartbeatPingConfig controls heartbeat ping endpoint behavior.
type HeartbeatPingConfig struct {
	TokenLimiter ratelimit.Limiter
}

// HeartbeatPingHandler handles GET/POST /heartbeat/{token}.
type HeartbeatPingHandler struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
	cfg    HeartbeatPingConfig
}

// NewHeartbeatPingHandler returns a handler for heartbeat ping requests.
func NewHeartbeatPingHandler(pool *pgxpool.Pool, logger *slog.Logger, cfg HeartbeatPingConfig) *HeartbeatPingHandler {
	return &HeartbeatPingHandler{
		pool:   pool,
		logger: logger,
		cfg:    cfg,
	}
}

type heartbeatPingResponse struct {
	Status string `json:"status"`
}

// ServeHTTP records an idempotent heartbeat ping for the monitor matching token.
func (h *HeartbeatPingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		WriteAPIError(w, http.StatusMethodNotAllowed, CodeMethodNotAllowed, "method not allowed")
		return
	}

	token := strings.TrimSpace(chi.URLParam(r, "token"))
	if token == "" {
		WriteAPIError(w, http.StatusNotFound, CodeNotFound, "not found")
		return
	}

	tokenHash := auth.HashPasswordResetToken(token)
	if h.cfg.TokenLimiter != nil && !h.cfg.TokenLimiter.Allow("heartbeat:"+tokenHash) {
		WriteAPIError(w, http.StatusTooManyRequests, CodeRateLimited, "too many heartbeat pings for this token")
		return
	}

	ctx := r.Context()
	queries := db.New(h.pool)

	monitor, err := queries.RecordHeartbeatPing(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteAPIError(w, http.StatusNotFound, CodeNotFound, "not found")
			return
		}
		h.logger.Error("record heartbeat ping failed", "error", err)
		WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
		return
	}

	h.logger.Info("heartbeat ping recorded",
		"monitor_id", monitor.ID,
		"token_prefix", monitor.Prefix,
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(heartbeatPingResponse{Status: monitor.Status})
}
