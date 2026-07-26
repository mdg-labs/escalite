package handlers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/mdg-labs/escalite/services/api/internal/audit"
	"github.com/mdg-labs/escalite/services/api/internal/auth"
	"github.com/mdg-labs/escalite/services/api/internal/db"
)

func authenticateForLogin(ctx context.Context, queries db.Querier, email, password string) (db.User, error) {
	account, err := auth.AuthenticateAccount(ctx, queries, email, password)
	if err != nil {
		return db.User{}, err
	}

	return auth.LoginMembership(ctx, queries, account.ID)
}

func createLoginSession(
	ctx context.Context,
	queries db.Querier,
	w http.ResponseWriter,
	user db.User,
	userAgent string,
	auditRecorder *audit.Recorder,
	requestMeta audit.RequestMeta,
) error {
	sessionID := uuid.Must(uuid.NewV7())
	expiresAt := time.Now().UTC().Add(auth.DefaultSessionTTL)

	_, err := queries.CreateSession(ctx, db.CreateSessionParams{
		ID:             sessionID,
		UserID:         user.ID,
		OrganizationID: user.OrganizationID,
		ExpiresAt:      pgtype.Timestamptz{Time: expiresAt, Valid: true},
		UserAgent:      pgtype.Text{String: userAgent, Valid: userAgent != ""},
	})
	if err != nil {
		return err
	}

	auth.SetSessionCookie(w, sessionID, expiresAt)
	auditRecorder.Login(ctx, queries, user.OrganizationID, user.ID, requestMeta)
	return nil
}

func mapAuthError(err error) (code, message string, status int, ok bool) {
	switch {
	case errors.Is(err, auth.ErrInvalidCredentials):
		return CodeUnauthenticated, invalidCredentialsMessage, 401, true
	case errors.Is(err, auth.ErrNoActiveMembership):
		return CodeUnauthenticated, invalidCredentialsMessage, 401, true
	case errors.Is(err, auth.ErrForbiddenOrg):
		return CodeForbidden, "access denied", 403, true
	default:
		if errors.Is(err, pgx.ErrNoRows) {
			return CodeUnauthenticated, invalidCredentialsMessage, 401, true
		}
		return "", "", 0, false
	}
}
