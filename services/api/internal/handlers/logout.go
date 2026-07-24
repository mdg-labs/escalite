package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mdg-labs/escalite/services/api/internal/audit"
	"github.com/mdg-labs/escalite/services/api/internal/auth"
	"github.com/mdg-labs/escalite/services/api/internal/db"
)

// LogoutHandler handles POST /api/v1/logout.
type LogoutHandler struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
	audit  *audit.Recorder
}

// NewLogoutHandler returns a handler that revokes the current session.
func NewLogoutHandler(pool *pgxpool.Pool, logger *slog.Logger) *LogoutHandler {
	return &LogoutHandler{pool: pool, logger: logger, audit: audit.NewRecorder(logger)}
}

func (h *LogoutHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteAPIError(w, http.StatusMethodNotAllowed, CodeMethodNotAllowed, "method not allowed")
		return
	}

	ctx := r.Context()
	queries := db.New(h.pool)
	requestMeta := audit.RequestMetaFromHTTP(r)

	if sc, ok := auth.SessionFromContext(ctx); ok {
		err := queries.RevokeSession(ctx, db.RevokeSessionParams{
			ID:             sc.Session.ID,
			OrganizationID: sc.Session.OrganizationID,
		})
		if err != nil {
			h.logger.Error("revoke session failed", "error", err)
			WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
			return
		}
		h.audit.Logout(ctx, queries, sc.User.OrganizationID, sc.User.ID, requestMeta)
	} else if cookie, err := r.Cookie(auth.SessionCookieName); err == nil && cookie.Value != "" {
		sessionID, parseErr := uuid.Parse(cookie.Value)
		if parseErr == nil {
			session, lookupErr := queries.GetActiveSessionByID(ctx, sessionID)
			if lookupErr == nil {
				_ = queries.RevokeSession(ctx, db.RevokeSessionParams{
					ID:             session.ID,
					OrganizationID: session.OrganizationID,
				})
				user, userErr := queries.GetUserByID(ctx, db.GetUserByIDParams{
					ID:             session.UserID,
					OrganizationID: session.OrganizationID,
				})
				if userErr == nil {
					h.audit.Logout(ctx, queries, user.OrganizationID, user.ID, requestMeta)
				}
			} else if !errors.Is(lookupErr, pgx.ErrNoRows) {
				h.logger.Error("load session for logout failed", "error", lookupErr)
			}
		}
	}

	auth.ClearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}
