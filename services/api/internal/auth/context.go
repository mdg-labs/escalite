package auth

import (
	"context"

	"github.com/google/uuid"

	"github.com/mdg-labs/escalite/services/api/internal/db"
)

type contextKey int

const (
	sessionContextKey contextKey = iota
	teamContextKey
)

// SessionContext holds the authenticated session and user loaded by session middleware.
type SessionContext struct {
	Session      db.Session
	User         db.User
	RefreshToken *db.RefreshToken
}

// WithSessionContext stores the authenticated session on the request context.
func WithSessionContext(ctx context.Context, sc SessionContext) context.Context {
	return context.WithValue(ctx, sessionContextKey, sc)
}

// SessionFromContext returns the authenticated session context, if present.
func SessionFromContext(ctx context.Context) (SessionContext, bool) {
	sc, ok := ctx.Value(sessionContextKey).(SessionContext)
	return sc, ok
}

// UserIDFromContext returns the authenticated user's ID when a session is present.
func UserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	sc, ok := SessionFromContext(ctx)
	if !ok {
		return uuid.UUID{}, false
	}
	return sc.User.ID, true
}

// WithTeamContext stores the authorized team on the request context.
func WithTeamContext(ctx context.Context, team db.Team) context.Context {
	return context.WithValue(ctx, teamContextKey, team)
}

// TeamFromContext returns the team authorized by team-access middleware.
func TeamFromContext(ctx context.Context) (db.Team, bool) {
	team, ok := ctx.Value(teamContextKey).(db.Team)
	return team, ok
}
