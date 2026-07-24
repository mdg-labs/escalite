package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mdg-labs/escalite/services/api/internal/auth"
	"github.com/mdg-labs/escalite/services/api/internal/authz"
	"github.com/mdg-labs/escalite/services/api/internal/db"
)

// RequireTeamAccess verifies org+team RBAC for routes with a {teamID} path parameter.
// Admins may access any team in their org; members require team membership.
func RequireTeamAccess(pool *pgxpool.Pool, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sc, ok := auth.SessionFromContext(r.Context())
			if !ok {
				WriteAPIError(w, http.StatusUnauthorized, CodeUnauthenticated, "authentication required")
				return
			}

			teamID, err := uuid.Parse(chi.URLParam(r, "teamID"))
			if err != nil {
				WriteAPIError(w, http.StatusBadRequest, CodeValidation, "teamID is invalid")
				return
			}

			queries := db.New(pool)
			team, err := authz.CheckTeamAccess(r.Context(), queries, sc.User, teamID)
			if err != nil {
				switch {
				case errors.Is(err, authz.ErrForbidden):
					WriteAPIError(w, http.StatusForbidden, CodeForbidden, "access denied")
				case errors.Is(err, authz.ErrNotFound):
					WriteAPIError(w, http.StatusNotFound, CodeNotFound, "team not found")
				default:
					logger.Error("team access check failed", "error", err)
					WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
				}
				return
			}

			ctx := auth.WithTeamContext(r.Context(), team)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
