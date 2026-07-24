package graph

import (
	"context"
	"errors"
	"strings"
	"time"

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
	"github.com/mdg-labs/escalite/services/engine/escalationapi"
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

func (r *mutationResolver) SnoozeAlert(ctx context.Context, id string, durationMinutes int) (*model.Alert, error) {
	sc, err := requireAuthSession(ctx)
	if err != nil {
		return nil, err
	}

	alertID, err := parseUUIDField(id, "id")
	if err != nil {
		return nil, err
	}

	queries := db.New(r.pool)
	alert, service, err := r.loadAlertWithTeamAccess(ctx, queries, sc, alertID)
	if err != nil {
		return nil, err
	}
	_ = service

	if err := escalationapi.SnoozeAlert(
		ctx,
		r.pool,
		r.jobs,
		alertID,
		sc.User.OrganizationID,
		int32(durationMinutes),
	); err != nil {
		if isEscalationValidationError(err) {
			return nil, gqlerr.New(handlers.CodeValidation, err.Error())
		}
		r.logger.Error("snooze alert failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	snoozed, err := queries.GetAlertByID(ctx, db.GetAlertByIDParams{
		ID:             alertID,
		OrganizationID: sc.User.OrganizationID,
	})
	if err != nil {
		r.logger.Error("reload snoozed alert failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	state, err := escalation.ParseState(snoozed.EscalationState)
	if err != nil {
		r.logger.Error("parse escalation state failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}
	if state.NextEscalationAt != nil {
		r.audit.AlertEscalationSnoozed(
			ctx,
			queries,
			sc.User.OrganizationID,
			sc.User.ID,
			alert.ID,
			int32(durationMinutes),
			state.NextEscalationAt.UTC().Format(time.RFC3339),
		)
	}

	return alertFromDB(snoozed, nil), nil
}

func (r *mutationResolver) ReEscalateAlert(ctx context.Context, id string) (*model.Alert, error) {
	sc, err := requireAuthSession(ctx)
	if err != nil {
		return nil, err
	}

	alertID, err := parseUUIDField(id, "id")
	if err != nil {
		return nil, err
	}

	queries := db.New(r.pool)
	alert, service, err := r.loadAlertWithTeamAccess(ctx, queries, sc, alertID)
	if err != nil {
		return nil, err
	}
	_ = service

	if err := escalationapi.ReEscalateAlert(
		ctx,
		r.pool,
		r.jobs,
		alertID,
		sc.User.OrganizationID,
	); err != nil {
		if isEscalationValidationError(err) {
			return nil, gqlerr.New(handlers.CodeValidation, err.Error())
		}
		r.logger.Error("re-escalate alert failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	reEscalated, err := queries.GetAlertByID(ctx, db.GetAlertByIDParams{
		ID:             alertID,
		OrganizationID: sc.User.OrganizationID,
	})
	if err != nil {
		r.logger.Error("reload re-escalated alert failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	r.audit.AlertReEscalated(ctx, queries, sc.User.OrganizationID, sc.User.ID, alert.ID)

	return alertFromDB(reEscalated, nil), nil
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
