package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mdg-labs/escalite/services/api/internal/auth"
	"github.com/mdg-labs/escalite/services/api/internal/db"
)

// MobileAuthHandler handles mobile auth code issuance, exchange, and refresh.
type MobileAuthHandler struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// NewMobileAuthHandler returns handlers for the mobile auth flow.
func NewMobileAuthHandler(pool *pgxpool.Pool, logger *slog.Logger) *MobileAuthHandler {
	return &MobileAuthHandler{pool: pool, logger: logger}
}

type mobileAuthUser struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

type mobileAuthCodeResponse struct {
	Code string `json:"code"`
}

type mobileAuthExchangeRequest struct {
	Code string `json:"code"`
}

type mobileAuthExchangeResponse struct {
	RefreshToken string         `json:"refresh_token"`
	User         mobileAuthUser `json:"user"`
}

type mobileAuthRefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type mobileAuthRefreshResponse struct {
	User mobileAuthUser `json:"user"`
}

// IssueCode handles POST /api/v1/mobile/auth/code for an authenticated web session.
func (h *MobileAuthHandler) IssueCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteAPIError(w, http.StatusMethodNotAllowed, CodeMethodNotAllowed, "method not allowed")
		return
	}

	sc, ok := auth.SessionFromContext(r.Context())
	if !ok {
		WriteAPIError(w, http.StatusUnauthorized, CodeUnauthenticated, "authentication required")
		return
	}

	plaintext, hash, err := auth.NewMobileAuthCode()
	if err != nil {
		h.logger.Error("generate mobile auth code failed", "error", err)
		WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
		return
	}

	codeID := uuid.Must(uuid.NewV7())
	expiresAt := time.Now().UTC().Add(auth.DefaultMobileAuthCodeTTL)

	ctx := r.Context()
	queries := db.New(h.pool)
	_, err = queries.CreateMobileAuthCode(ctx, db.CreateMobileAuthCodeParams{
		ID:             codeID,
		UserID:         sc.User.ID,
		OrganizationID: sc.User.OrganizationID,
		CodeHash:       hash,
		ExpiresAt:      pgtype.Timestamptz{Time: expiresAt, Valid: true},
	})
	if err != nil {
		h.logger.Error("create mobile auth code failed", "error", err)
		WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(mobileAuthCodeResponse{Code: plaintext})
}

// Exchange handles POST /api/v1/mobile/auth/exchange for a one-time auth code.
func (h *MobileAuthHandler) Exchange(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteAPIError(w, http.StatusMethodNotAllowed, CodeMethodNotAllowed, "method not allowed")
		return
	}

	var req mobileAuthExchangeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteAPIError(w, http.StatusBadRequest, CodeValidation, "invalid JSON body")
		return
	}

	code := strings.TrimSpace(req.Code)
	if code == "" {
		WriteAPIError(w, http.StatusBadRequest, CodeValidation, "code is required")
		return
	}

	ctx := r.Context()
	queries := db.New(h.pool)
	codeHash := auth.HashMobileAuthCode(code)

	authCode, err := queries.GetMobileAuthCodeByHash(ctx, codeHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteAPIError(w, http.StatusUnauthorized, CodeUnauthenticated, "invalid or expired code")
			return
		}
		h.logger.Error("lookup mobile auth code failed", "error", err)
		WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
		return
	}

	if err := queries.MarkMobileAuthCodeUsed(ctx, db.MarkMobileAuthCodeUsedParams{
		ID:             authCode.ID,
		OrganizationID: authCode.OrganizationID,
	}); err != nil {
		h.logger.Error("mark mobile auth code used failed", "error", err)
		WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
		return
	}

	user, err := queries.GetUserByID(ctx, db.GetUserByIDParams{
		ID:             authCode.UserID,
		OrganizationID: authCode.OrganizationID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteAPIError(w, http.StatusUnauthorized, CodeUnauthenticated, "invalid or expired code")
			return
		}
		h.logger.Error("load user for mobile exchange failed", "error", err)
		WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
		return
	}

	refreshPlaintext, refreshHash, err := auth.NewRefreshToken()
	if err != nil {
		h.logger.Error("generate refresh token failed", "error", err)
		WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
		return
	}

	refreshID := uuid.Must(uuid.NewV7())
	expiresAt := time.Now().UTC().Add(auth.DefaultRefreshTokenTTL)
	userAgent := r.UserAgent()

	_, err = queries.CreateRefreshToken(ctx, db.CreateRefreshTokenParams{
		ID:             refreshID,
		UserID:         user.ID,
		OrganizationID: user.OrganizationID,
		TokenHash:      refreshHash,
		UserAgent:      pgtype.Text{String: userAgent, Valid: userAgent != ""},
		ExpiresAt:      pgtype.Timestamptz{Time: expiresAt, Valid: true},
	})
	if err != nil {
		h.logger.Error("create refresh token failed", "error", err)
		WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(mobileAuthExchangeResponse{
		RefreshToken: refreshPlaintext,
		User: mobileAuthUser{
			ID:    user.ID.String(),
			Email: user.Email,
			Role:  user.Role,
		},
	})
}

// Refresh handles POST /api/v1/mobile/auth/refresh for a device refresh token.
func (h *MobileAuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteAPIError(w, http.StatusMethodNotAllowed, CodeMethodNotAllowed, "method not allowed")
		return
	}

	var req mobileAuthRefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteAPIError(w, http.StatusBadRequest, CodeValidation, "invalid JSON body")
		return
	}

	refreshToken := strings.TrimSpace(req.RefreshToken)
	if refreshToken == "" {
		if bearer, ok := auth.ParseBearerToken(r.Header.Get("Authorization")); ok {
			refreshToken = bearer
		}
	}
	if refreshToken == "" {
		WriteAPIError(w, http.StatusBadRequest, CodeValidation, "refresh_token is required")
		return
	}

	ctx := r.Context()
	queries := db.New(h.pool)
	tokenHash := auth.HashRefreshToken(refreshToken)

	stored, err := queries.GetRefreshTokenByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteAPIError(w, http.StatusUnauthorized, CodeUnauthenticated, "invalid or revoked refresh token")
			return
		}
		h.logger.Error("lookup refresh token failed", "error", err)
		WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
		return
	}

	user, err := queries.GetUserByID(ctx, db.GetUserByIDParams{
		ID:             stored.UserID,
		OrganizationID: stored.OrganizationID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteAPIError(w, http.StatusUnauthorized, CodeUnauthenticated, "invalid or revoked refresh token")
			return
		}
		h.logger.Error("load user for mobile refresh failed", "error", err)
		WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(mobileAuthRefreshResponse{
		User: mobileAuthUser{
			ID:    user.ID.String(),
			Email: user.Email,
			Role:  user.Role,
		},
	})
}
