package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mdg-labs/escalite/services/api/internal/audit"
	"github.com/mdg-labs/escalite/services/api/internal/auth"
	"github.com/mdg-labs/escalite/services/api/internal/db"
)

const minPasswordLength = 8

// SetupHandler handles first-admin bootstrap when no users exist.
type SetupHandler struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
	audit  *audit.Recorder
}

// NewSetupHandler returns a handler for POST /api/v1/setup.
func NewSetupHandler(pool *pgxpool.Pool, logger *slog.Logger) *SetupHandler {
	return &SetupHandler{pool: pool, logger: logger, audit: audit.NewRecorder(logger)}
}

type setupRequest struct {
	OrganizationName string `json:"organizationName"`
	Email            string `json:"email"`
	Password         string `json:"password"`
}

type setupOrganization struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type setupUser struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

type setupResponse struct {
	Organization setupOrganization `json:"organization"`
	User         setupUser         `json:"user"`
}

func (h *SetupHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteAPIError(w, http.StatusMethodNotAllowed, CodeMethodNotAllowed, "method not allowed")
		return
	}

	var req setupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteAPIError(w, http.StatusBadRequest, CodeValidation, "invalid JSON body")
		return
	}

	orgName := strings.TrimSpace(req.OrganizationName)
	email := strings.ToLower(strings.TrimSpace(req.Email))
	password := req.Password

	if orgName == "" {
		WriteAPIError(w, http.StatusBadRequest, CodeValidation, "organizationName is required")
		return
	}
	if email == "" {
		WriteAPIError(w, http.StatusBadRequest, CodeValidation, "email is required")
		return
	}
	if _, err := mail.ParseAddress(email); err != nil {
		WriteAPIError(w, http.StatusBadRequest, CodeValidation, "email is invalid")
		return
	}
	if len(password) < minPasswordLength {
		WriteAPIError(w, http.StatusBadRequest, CodeValidation, "password must be at least 8 characters")
		return
	}

	ctx := r.Context()
	queries := db.New(h.pool)

	userCount, err := queries.CountUsers(ctx)
	if err != nil {
		h.logger.Error("count users failed", "error", err)
		WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
		return
	}
	if userCount > 0 {
		WriteAPIError(w, http.StatusForbidden, CodeForbidden, "setup is disabled after the first admin exists")
		return
	}

	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		h.logger.Error("hash password failed", "error", err)
		WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
		return
	}

	orgID := uuid.Must(uuid.NewV7())
	userID := uuid.Must(uuid.NewV7())
	sessionID := uuid.Must(uuid.NewV7())
	expiresAt := time.Now().UTC().Add(auth.DefaultSessionTTL)
	userAgent := r.UserAgent()

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		h.logger.Error("begin transaction failed", "error", err)
		WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
		return
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	txQueries := queries.WithTx(tx)

	bootstrap, err := txQueries.BootstrapOrganizationWithAdmin(ctx, db.BootstrapOrganizationWithAdminParams{
		OrgID:        orgID,
		OrgName:      orgName,
		UserID:       userID,
		Email:        email,
		PasswordHash: pgtype.Text{String: passwordHash, Valid: true},
	})
	if err != nil {
		h.logger.Error("bootstrap organization failed", "error", err)
		WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
		return
	}

	_, err = txQueries.CreateSession(ctx, db.CreateSessionParams{
		ID:             sessionID,
		UserID:         userID,
		OrganizationID: orgID,
		ExpiresAt:      pgtype.Timestamptz{Time: expiresAt, Valid: true},
		UserAgent:      pgtype.Text{String: userAgent, Valid: userAgent != ""},
	})
	if err != nil {
		h.logger.Error("create session failed", "error", err)
		WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
		return
	}

	requestMeta := audit.RequestMetaFromHTTP(r)
	h.audit.Login(ctx, txQueries, orgID, userID, requestMeta)

	if err := tx.Commit(ctx); err != nil {
		h.logger.Error("commit transaction failed", "error", err)
		WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
		return
	}

	auth.SetSessionCookie(w, sessionID, expiresAt)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(setupResponse{
		Organization: setupOrganization{
			ID:   bootstrap.OrganizationID.String(),
			Name: bootstrap.OrganizationName,
		},
		User: setupUser{
			ID:    bootstrap.UserID.String(),
			Email: bootstrap.UserEmail,
			Role:  bootstrap.UserRole,
		},
	})
}
