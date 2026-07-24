package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mdg-labs/escalite/services/api/internal/auth"
	"github.com/mdg-labs/escalite/services/api/internal/db"
)

// RequireSession validates the session cookie and attaches the session and user to the request context.
// Unauthenticated or revoked sessions receive HTTP 401 with code UNAUTHENTICATED (GraphQL-equivalent semantics).
func RequireSession(pool *pgxpool.Pool, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(auth.SessionCookieName)
			if err != nil || cookie.Value == "" {
				WriteAPIError(w, http.StatusUnauthorized, CodeUnauthenticated, "authentication required")
				return
			}

			sessionID, err := uuid.Parse(cookie.Value)
			if err != nil {
				WriteAPIError(w, http.StatusUnauthorized, CodeUnauthenticated, "authentication required")
				return
			}

			ctx := r.Context()
			queries := db.New(pool)

			session, err := queries.GetActiveSessionByID(ctx, sessionID)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					WriteAPIError(w, http.StatusUnauthorized, CodeUnauthenticated, "authentication required")
					return
				}
				logger.Error("load session failed", "error", err)
				WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
				return
			}

			user, err := queries.GetUserByID(ctx, db.GetUserByIDParams{
				ID:             session.UserID,
				OrganizationID: session.OrganizationID,
			})
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					WriteAPIError(w, http.StatusUnauthorized, CodeUnauthenticated, "authentication required")
					return
				}
				logger.Error("load session user failed", "error", err)
				WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
				return
			}

			ctx = auth.WithSessionContext(ctx, auth.SessionContext{
				Session: session,
				User:    user,
			})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
