package graph

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/mdg-labs/escalite/services/api/graph/model"
	"github.com/mdg-labs/escalite/services/api/internal/auth"
	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/api/internal/gqlerr"
	"github.com/mdg-labs/escalite/services/api/internal/handlers"
	"github.com/mdg-labs/escalite/services/api/internal/realtime"
)

func requireOrgSubscriptionAccess(ctx context.Context, orgID string) (auth.SessionContext, uuid.UUID, error) {
	sc, err := requireAuthSession(ctx)
	if err != nil {
		return auth.SessionContext{}, uuid.Nil, err
	}

	orgUUID, err := uuid.Parse(orgID)
	if err != nil {
		return auth.SessionContext{}, uuid.Nil, gqlerr.New(handlers.CodeValidation, "invalid organization id")
	}
	if orgUUID != sc.User.OrganizationID {
		return auth.SessionContext{}, uuid.Nil, gqlerr.New(handlers.CodeForbidden, "access denied")
	}

	return sc, orgUUID, nil
}

func (r *Resolver) loadAlertForSubscription(ctx context.Context, alertID, orgID uuid.UUID) (*model.Alert, error) {
	queries := db.New(r.pool)

	alert, err := queries.GetAlertByID(ctx, db.GetAlertByIDParams{
		ID:             alertID,
		OrganizationID: orgID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, gqlerr.New(handlers.CodeNotFound, "alert not found")
		}
		r.logger.Error("load subscription alert failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	var acknowledgedBy *db.User
	if alert.AcknowledgedByUserID.Valid {
		user, err := queries.GetUserByID(ctx, db.GetUserByIDParams{
			ID:             alert.AcknowledgedByUserID.Bytes,
			OrganizationID: orgID,
		})
		if err != nil {
			r.logger.Error("load subscription alert acknowledger failed", "error", err)
			return nil, gqlerr.New(handlers.CodeInternal, "internal error")
		}
		acknowledgedBy = &user
	}

	return alertFromDB(alert, acknowledgedBy), nil
}

func (r *Resolver) streamAlertUpdates(
	ctx context.Context,
	orgID uuid.UUID,
) (<-chan *model.Alert, error) {
	if r.realtime == nil {
		ch := make(chan *model.Alert)
		close(ch)
		return ch, nil
	}

	events, cancel := r.realtime.SubscribeAlerts(orgID)
	out := make(chan *model.Alert)

	go func() {
		defer cancel()
		defer close(out)

		for {
			select {
			case <-ctx.Done():
				return
			case evt, ok := <-events:
				if !ok {
					return
				}
				alert, err := r.loadAlertForSubscription(ctx, evt.AlertID, orgID)
				if err != nil {
					r.logger.Warn("subscription alert update skipped", "error", err, "alert_id", evt.AlertID)
					continue
				}
				select {
				case out <- alert:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return out, nil
}

func (r *Resolver) streamOnCallUpdates(
	ctx context.Context,
	orgID uuid.UUID,
) (<-chan *model.OnCallUpdatedEvent, error) {
	if r.realtime == nil {
		ch := make(chan *model.OnCallUpdatedEvent)
		close(ch)
		return ch, nil
	}

	events, cancel := r.realtime.SubscribeSchedules(orgID)
	out := make(chan *model.OnCallUpdatedEvent)

	go func() {
		defer cancel()
		defer close(out)

		for {
			select {
			case <-ctx.Done():
				return
			case evt, ok := <-events:
				if !ok {
					return
				}
				payload := onCallUpdatedEventFromRealtime(evt)
				select {
				case out <- payload:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return out, nil
}

func onCallUpdatedEventFromRealtime(evt realtime.ScheduleEvent) *model.OnCallUpdatedEvent {
	return &model.OnCallUpdatedEvent{
		ScheduleID:     evt.ScheduleID.String(),
		OrganizationID: evt.OrganizationID.String(),
	}
}

func (r *Resolver) loadTimelineEventForSubscription(
	ctx context.Context,
	timelineEventID, incidentID, orgID uuid.UUID,
) (*model.TimelineEvent, error) {
	queries := db.New(r.pool)

	event, err := queries.GetTimelineEventByID(ctx, db.GetTimelineEventByIDParams{
		ID:             timelineEventID,
		OrganizationID: orgID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, gqlerr.New(handlers.CodeNotFound, "timeline event not found")
		}
		r.logger.Error("load subscription timeline event failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}
	if event.IncidentID != incidentID {
		return nil, gqlerr.New(handlers.CodeNotFound, "timeline event not found")
	}

	result := timelineEventFromDB(event)
	if event.ActorID.Valid {
		user, err := queries.GetUserByID(ctx, db.GetUserByIDParams{
			ID:             event.ActorID.Bytes,
			OrganizationID: orgID,
		})
		if err != nil {
			r.logger.Error("load subscription timeline actor failed", "error", err)
			return nil, gqlerr.New(handlers.CodeInternal, "internal error")
		}
		result.Actor = userFromDB(user)
	}

	return result, nil
}

func (r *Resolver) streamIncidentTimelineUpdates(
	ctx context.Context,
	incidentID uuid.UUID,
	orgID uuid.UUID,
) (<-chan *model.TimelineEvent, error) {
	if r.realtime == nil {
		ch := make(chan *model.TimelineEvent)
		close(ch)
		return ch, nil
	}

	events, cancel := r.realtime.SubscribeTimeline(incidentID)
	out := make(chan *model.TimelineEvent)

	go func() {
		defer cancel()
		defer close(out)

		for {
			select {
			case <-ctx.Done():
				return
			case evt, ok := <-events:
				if !ok {
					return
				}
				timelineEvent, err := r.loadTimelineEventForSubscription(ctx, evt.TimelineEventID, incidentID, orgID)
				if err != nil {
					r.logger.Warn("subscription timeline update skipped", "error", err, "timeline_event_id", evt.TimelineEventID)
					continue
				}
				select {
				case out <- timelineEvent:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return out, nil
}

func requireIncidentSubscriptionAccess(
	ctx context.Context,
	r *Resolver,
	incidentID string,
) (auth.SessionContext, uuid.UUID, uuid.UUID, error) {
	sc, err := requireAuthSession(ctx)
	if err != nil {
		return auth.SessionContext{}, uuid.Nil, uuid.Nil, err
	}

	incidentUUID, err := uuid.Parse(incidentID)
	if err != nil {
		return auth.SessionContext{}, uuid.Nil, uuid.Nil, gqlerr.New(handlers.CodeValidation, "invalid incident id")
	}

	queries := db.New(r.pool)
	if _, err := r.loadIncidentWithTeamAccess(ctx, queries, sc, incidentUUID); err != nil {
		return auth.SessionContext{}, uuid.Nil, uuid.Nil, err
	}

	return sc, incidentUUID, sc.User.OrganizationID, nil
}
