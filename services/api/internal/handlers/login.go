package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/mail"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mdg-labs/escalite/services/api/internal/audit"
	"github.com/mdg-labs/escalite/services/api/internal/auth"
	"github.com/mdg-labs/escalite/services/api/internal/db"
)

const invalidCredentialsMessage = "invalid credentials"

// LoginHandler handles POST /api/v1/login.
type LoginHandler struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
	audit  *audit.Recorder
}

// NewLoginHandler returns a handler for email+password login.
func NewLoginHandler(pool *pgxpool.Pool, logger *slog.Logger) *LoginHandler {
	return &LoginHandler{pool: pool, logger: logger, audit: audit.NewRecorder(logger)}
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginUser struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

type loginResponse struct {
	User loginUser `json:"user"`
}

func (h *LoginHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteAPIError(w, http.StatusMethodNotAllowed, CodeMethodNotAllowed, "method not allowed")
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteAPIError(w, http.StatusBadRequest, CodeValidation, "invalid JSON body")
		return
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	password := req.Password

	if email == "" {
		WriteAPIError(w, http.StatusBadRequest, CodeValidation, "email is required")
		return
	}
	if _, err := mail.ParseAddress(email); err != nil {
		WriteAPIError(w, http.StatusBadRequest, CodeValidation, "email is invalid")
		return
	}
	if password == "" {
		WriteAPIError(w, http.StatusBadRequest, CodeValidation, "password is required")
		return
	}

	ctx := r.Context()
	queries := db.New(h.pool)
	requestMeta := audit.RequestMetaFromHTTP(r)
	requestMeta.Email = email

	user, err := authenticateForLogin(ctx, queries, email, password)
	if err != nil {
		if code, message, status, ok := mapAuthError(err); ok {
			if errors.Is(err, auth.ErrInvalidCredentials) || errors.Is(err, auth.ErrNoActiveMembership) {
				var failedUser *db.User
				if account, accountErr := queries.GetAccountByEmail(ctx, email); accountErr == nil {
					if membership, membershipErr := auth.LoginMembership(ctx, queries, account.ID); membershipErr == nil {
						failedUser = &membership
					}
				}
				h.recordFailedLogin(ctx, queries, failedUser, requestMeta)
			}
			WriteAPIError(w, status, code, message)
			return
		}
		h.logger.Error("login failed", "error", err)
		WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
		return
	}

	if err := createLoginSession(ctx, queries, w, user, r.UserAgent(), h.audit, requestMeta); err != nil {
		h.logger.Error("create session failed", "error", err)
		WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(loginResponse{
		User: loginUser{
			ID:    user.ID.String(),
			Email: user.Email,
			Role:  user.Role,
		},
	})
}

func (h *LoginHandler) recordFailedLogin(ctx context.Context, queries db.Querier, user *db.User, meta audit.RequestMeta) {
	orgID := uuid.Nil
	var targetUserID *uuid.UUID
	if user != nil {
		orgID = user.OrganizationID
		targetUserID = &user.ID
	} else {
		org, err := queries.GetFirstOrganization(ctx)
		if err == nil {
			orgID = org.ID
		}
	}
	h.audit.LoginFailed(ctx, queries, orgID, targetUserID, meta)
}
