package graph

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/mdg-labs/escalite/services/api/internal/auth"
	"github.com/mdg-labs/escalite/services/api/internal/authz"
	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/api/internal/gqlerr"
	"github.com/mdg-labs/escalite/services/api/internal/handlers"
)

func requireAuthSession(ctx context.Context) (auth.SessionContext, error) {
	sc, ok := auth.SessionFromContext(ctx)
	if !ok {
		return auth.SessionContext{}, gqlerr.New(handlers.CodeUnauthenticated, "authentication required")
	}
	return sc, nil
}

func (r *mutationResolver) loadAlertWithTeamAccess(
	ctx context.Context,
	queries *db.Queries,
	sc auth.SessionContext,
	alertID uuid.UUID,
) (db.Alert, db.Service, error) {
	alert, err := queries.GetAlertByID(ctx, db.GetAlertByIDParams{
		ID:             alertID,
		OrganizationID: sc.User.OrganizationID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.Alert{}, db.Service{}, gqlerr.New(handlers.CodeNotFound, "alert not found")
		}
		r.logger.Error("load alert failed", "error", err)
		return db.Alert{}, db.Service{}, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	service, err := queries.GetServiceByID(ctx, db.GetServiceByIDParams{
		ID:             alert.ServiceID,
		OrganizationID: sc.User.OrganizationID,
	})
	if err != nil {
		r.logger.Error("load service failed", "error", err)
		return db.Alert{}, db.Service{}, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	if _, err := authz.CheckTeamAccess(ctx, queries, sc.User, service.TeamID); err != nil {
		if errors.Is(err, authz.ErrNotFound) || errors.Is(err, authz.ErrForbidden) {
			return db.Alert{}, db.Service{}, gqlerr.New(handlers.CodeForbidden, "access denied")
		}
		r.logger.Error("check team access failed", "error", err)
		return db.Alert{}, db.Service{}, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	return alert, service, nil
}

func isEscalationValidationError(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "must be positive") ||
		strings.Contains(msg, "exceeds maximum") ||
		strings.Contains(msg, "only triggered alerts can be snoozed") ||
		strings.Contains(msg, "escalation is exhausted") ||
		strings.Contains(msg, "no active escalation step") ||
		strings.Contains(msg, "cannot re-escalate a closed alert")
}
