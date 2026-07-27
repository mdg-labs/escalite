package graph

import (
	"context"
	"time"

	"github.com/mdg-labs/escalite/services/api/graph/model"
	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/api/internal/escalation"
	"github.com/mdg-labs/escalite/services/api/internal/gqlerr"
	"github.com/mdg-labs/escalite/services/api/internal/handlers"
)

func (r *alertResolver) Service(ctx context.Context, obj *model.Alert) (*model.Service, error) {
	if obj == nil {
		return nil, nil
	}

	sc, err := requireAuthSession(ctx)
	if err != nil {
		return nil, err
	}

	alertID, err := parseUUIDField(obj.ID, "id")
	if err != nil {
		return nil, err
	}

	queries := db.New(r.pool)
	_, service, err := r.loadAlertWithTeamAccess(ctx, queries, sc, alertID)
	if err != nil {
		return nil, err
	}

	return serviceFromDB(service), nil
}

func (r *alertResolver) EscalationState(ctx context.Context, obj *model.Alert) (*model.AlertEscalationState, error) {
	if obj == nil {
		return nil, nil
	}

	sc, err := requireAuthSession(ctx)
	if err != nil {
		return nil, err
	}

	alertID, err := parseUUIDField(obj.ID, "id")
	if err != nil {
		return nil, err
	}

	queries := db.New(r.pool)
	alert, _, err := r.loadAlertWithTeamAccess(ctx, queries, sc, alertID)
	if err != nil {
		return nil, err
	}

	state, err := escalation.ParseState(alert.EscalationState)
	if err != nil {
		r.logger.Error("parse alert escalation state failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	if state.CurrentStep == 0 && state.NextEscalationAt == nil && !state.EscalatedExhausted {
		return nil, nil
	}

	return alertEscalationStateFromEngine(state), nil
}

func (r *alertResolver) NotificationAttempts(ctx context.Context, obj *model.Alert) ([]*model.NotificationAttempt, error) {
	if obj == nil {
		return nil, nil
	}

	sc, err := requireAuthSession(ctx)
	if err != nil {
		return nil, err
	}

	alertID, err := parseUUIDField(obj.ID, "id")
	if err != nil {
		return nil, err
	}

	queries := db.New(r.pool)
	if _, _, err := r.loadAlertWithTeamAccess(ctx, queries, sc, alertID); err != nil {
		return nil, err
	}

	rows, err := queries.ListNotificationAttemptsByAlertID(ctx, db.ListNotificationAttemptsByAlertIDParams{
		AlertID:        alertID,
		OrganizationID: sc.User.OrganizationID,
	})
	if err != nil {
		r.logger.Error("list notification attempts failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	result := make([]*model.NotificationAttempt, 0, len(rows))
	for _, row := range rows {
		result = append(result, notificationAttemptFromDB(row))
	}
	return result, nil
}

func alertEscalationStateFromEngine(state escalation.State) *model.AlertEscalationState {
	var nextEscalationAt *time.Time
	if state.NextEscalationAt != nil {
		t := state.NextEscalationAt.UTC()
		nextEscalationAt = &t
	}

	return &model.AlertEscalationState{
		CurrentStep:        state.CurrentStep,
		NextEscalationAt:   nextEscalationAt,
		EscalatedExhausted: state.EscalatedExhausted,
	}
}

func notificationAttemptFromDB(attempt db.NotificationAttempt) *model.NotificationAttempt {
	var sentAt *time.Time
	if attempt.SentAt.Valid {
		t := attempt.SentAt.Time.UTC()
		sentAt = &t
	}

	return &model.NotificationAttempt{
		ID:        attempt.ID.String(),
		Channel:   attempt.Channel,
		Status:    attempt.Status,
		SentAt:    sentAt,
		CreatedAt: attempt.CreatedAt.Time.UTC(),
	}
}
