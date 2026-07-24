package graph

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/mdg-labs/escalite/services/api/graph/model"
	"github.com/mdg-labs/escalite/services/api/internal/auth"
	"github.com/mdg-labs/escalite/services/api/internal/authz"
	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/api/internal/escalation"
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

func (r *mutationResolver) AcknowledgeAlert(ctx context.Context, id string) (*model.Alert, error) {
	sc, err := requireAuthSession(ctx)
	if err != nil {
		return nil, err
	}

	alertID, err := parseUUIDField(id, "id")
	if err != nil {
		return nil, err
	}

	queries := db.New(r.pool)
	alert, err := queries.GetAlertByID(ctx, db.GetAlertByIDParams{
		ID:             alertID,
		OrganizationID: sc.User.OrganizationID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, gqlerr.New(handlers.CodeNotFound, "alert not found")
		}
		r.logger.Error("load alert failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	if alert.Status == "closed" {
		return nil, gqlerr.New(handlers.CodeValidation, "cannot acknowledge a closed alert")
	}
	if alert.Status == "acknowledged" {
		return nil, gqlerr.New(handlers.CodeValidation, "alert is already acknowledged")
	}

	service, err := queries.GetServiceByID(ctx, db.GetServiceByIDParams{
		ID:             alert.ServiceID,
		OrganizationID: sc.User.OrganizationID,
	})
	if err != nil {
		r.logger.Error("load service failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	if _, err := authz.CheckTeamAccess(ctx, queries, sc.User, service.TeamID); err != nil {
		if errors.Is(err, authz.ErrNotFound) || errors.Is(err, authz.ErrForbidden) {
			return nil, gqlerr.New(handlers.CodeForbidden, "access denied")
		}
		r.logger.Error("check team access failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	state, err := escalation.ParseState(alert.EscalationState)
	if err != nil {
		r.logger.Error("parse escalation state failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}
	state = escalation.ClearedTimerState(state)

	raw, err := escalation.MarshalState(state)
	if err != nil {
		r.logger.Error("marshal escalation state failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	acknowledged, err := queries.AcknowledgeAlert(ctx, db.AcknowledgeAlertParams{
		ID:                   alertID,
		OrganizationID:       sc.User.OrganizationID,
		EscalationState:      raw,
		AcknowledgedByUserID: pgtype.UUID{Bytes: sc.User.ID, Valid: true},
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, gqlerr.New(handlers.CodeValidation, "cannot acknowledge alert in its current state")
		}
		r.logger.Error("acknowledge alert failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	return alertFromDB(acknowledged, &sc.User), nil
}

func (r *mutationResolver) CloseAlert(ctx context.Context, id string) (*model.Alert, error) {
	sc, err := requireAuthSession(ctx)
	if err != nil {
		return nil, err
	}

	alertID, err := parseUUIDField(id, "id")
	if err != nil {
		return nil, err
	}

	queries := db.New(r.pool)
	alert, err := queries.GetAlertByID(ctx, db.GetAlertByIDParams{
		ID:             alertID,
		OrganizationID: sc.User.OrganizationID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, gqlerr.New(handlers.CodeNotFound, "alert not found")
		}
		r.logger.Error("load alert failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	if alert.Status == "closed" {
		return nil, gqlerr.New(handlers.CodeValidation, "alert is already closed")
	}

	service, err := queries.GetServiceByID(ctx, db.GetServiceByIDParams{
		ID:             alert.ServiceID,
		OrganizationID: sc.User.OrganizationID,
	})
	if err != nil {
		r.logger.Error("load service failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	if _, err := authz.CheckTeamAccess(ctx, queries, sc.User, service.TeamID); err != nil {
		if errors.Is(err, authz.ErrNotFound) || errors.Is(err, authz.ErrForbidden) {
			return nil, gqlerr.New(handlers.CodeForbidden, "access denied")
		}
		r.logger.Error("check team access failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	state, err := escalation.ParseState(alert.EscalationState)
	if err != nil {
		r.logger.Error("parse escalation state failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}
	state = escalation.ClearedTimerState(state)

	raw, err := escalation.MarshalState(state)
	if err != nil {
		r.logger.Error("marshal escalation state failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	closed, err := queries.CloseAlert(ctx, db.CloseAlertParams{
		ID:              alertID,
		OrganizationID:  sc.User.OrganizationID,
		EscalationState: raw,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, gqlerr.New(handlers.CodeValidation, "cannot close alert in its current state")
		}
		r.logger.Error("close alert failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	var acknowledgedBy *db.User
	if closed.AcknowledgedByUserID.Valid {
		user, err := queries.GetUserByID(ctx, db.GetUserByIDParams{
			ID:             uuid.UUID(closed.AcknowledgedByUserID.Bytes),
			OrganizationID: sc.User.OrganizationID,
		})
		if err == nil {
			acknowledgedBy = &user
		}
	}

	return alertFromDB(closed, acknowledgedBy), nil
}
