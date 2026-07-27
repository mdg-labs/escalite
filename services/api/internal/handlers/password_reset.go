package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/api/internal/email"
)

// PasswordResetHandler handles password reset request and confirm endpoints.
type PasswordResetHandler struct {
	service *PasswordResetService
	logger  *slog.Logger
}

// NewPasswordResetHandler returns handlers for password reset flows.
func NewPasswordResetHandler(
	pool *pgxpool.Pool,
	logger *slog.Logger,
	mail email.Sender,
	cfg PasswordResetConfig,
) *PasswordResetHandler {
	return &PasswordResetHandler{
		service: NewPasswordResetService(pool, db.New(pool), mail, cfg, logger),
		logger:  logger,
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

	clientIP := clientIPFromRequest(r)
	message, err := h.service.RequestReset(r.Context(), req.Email, clientIP)
	if err != nil {
		writePasswordResetError(w, err)
		return
	}

	h.writeAccepted(w, message)
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

	if err := h.service.ConfirmReset(r.Context(), req.Token, req.Password); err != nil {
		writePasswordResetError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func clientIPFromRequest(r *http.Request) string {
	return ClientIPFromRequest(r)
}

// ClientIPFromRequest returns the best-effort client IP for rate limiting.
func ClientIPFromRequest(r *http.Request) string {
	clientIP := strings.TrimSpace(r.RemoteAddr)
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
		clientIP = strings.TrimSpace(strings.Split(forwarded, ",")[0])
	}
	return clientIP
}

func writePasswordResetError(w http.ResponseWriter, err error) {
	var resetErr *PasswordResetError
	if !errors.As(err, &resetErr) {
		WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
		return
	}

	status := http.StatusInternalServerError
	switch resetErr.Code {
	case CodeValidation:
		status = http.StatusBadRequest
	case CodeRateLimited:
		status = http.StatusTooManyRequests
	}
	WriteAPIError(w, status, resetErr.Code, resetErr.Message)
}

func (h *PasswordResetHandler) writeAccepted(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(passwordResetAcceptedResponse{
		Message: message,
	})
}
