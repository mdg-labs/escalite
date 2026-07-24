package handlers

import (
	"context"
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
			ctx, ok := attachSession(r.Context(), r, pool, logger, w)
			if !ok {
				return
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// AttachSession loads a valid session into context when present but allows unauthenticated requests.
func AttachSession(pool *pgxpool.Pool, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, _ := attachSessionOptional(r.Context(), r, pool, logger)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func attachSession(
	ctx context.Context,
	r *http.Request,
	pool *pgxpool.Pool,
	logger *slog.Logger,
	w http.ResponseWriter,
) (context.Context, bool) {
	sc, ok, err := loadSessionContext(ctx, r, pool, logger)
	if err != nil {
		WriteAPIError(w, http.StatusInternalServerError, CodeInternal, "internal error")
		return ctx, false
	}
	if !ok {
		WriteAPIError(w, http.StatusUnauthorized, CodeUnauthenticated, "authentication required")
		return ctx, false
	}
	return auth.WithSessionContext(ctx, sc), true
}

func attachSessionOptional(
	ctx context.Context,
	r *http.Request,
	pool *pgxpool.Pool,
	logger *slog.Logger,
) (context.Context, bool) {
	sc, ok, err := loadSessionContext(ctx, r, pool, logger)
	if err != nil || !ok {
		return ctx, false
	}
	return auth.WithSessionContext(ctx, sc), true
}

func loadSessionContext(
	ctx context.Context,
	r *http.Request,
	pool *pgxpool.Pool,
	logger *slog.Logger,
) (auth.SessionContext, bool, error) {
	cookie, err := sessionCookie(ctx, r)
	if err != nil || cookie == nil || cookie.Value == "" {
		return auth.SessionContext{}, false, nil
	}

	sessionID, err := uuid.Parse(cookie.Value)
	if err != nil {
		return auth.SessionContext{}, false, nil
	}

	queries := db.New(pool)

	session, err := queries.GetActiveSessionByID(ctx, sessionID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return auth.SessionContext{}, false, nil
		}
		logger.Error("load session failed", "error", err)
		return auth.SessionContext{}, false, err
	}

	user, err := queries.GetUserByID(ctx, db.GetUserByIDParams{
		ID:             session.UserID,
		OrganizationID: session.OrganizationID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return auth.SessionContext{}, false, nil
		}
		logger.Error("load session user failed", "error", err)
		return auth.SessionContext{}, false, err
	}

	return auth.SessionContext{
		Session: session,
		User:    user,
	}, true, nil
}

func sessionCookie(ctx context.Context, r *http.Request) (*http.Cookie, error) {
	if r != nil {
		return r.Cookie(auth.SessionCookieName)
	}
	if req, ok := ctx.Value(requestContextKey).(*http.Request); ok && req != nil {
		return req.Cookie(auth.SessionCookieName)
	}
	return nil, http.ErrNoCookie
}

type requestContextKeyType struct{}

var requestContextKey = requestContextKeyType{}

// WithRequest stores the active HTTP request on the context for session loading.
func WithRequest(ctx context.Context, r *http.Request) context.Context {
	return context.WithValue(ctx, requestContextKey, r)
}

// WithRequestMiddleware stores the active HTTP request on the context for session loading.
func WithRequestMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r.WithContext(WithRequest(r.Context(), r)))
	})
}
