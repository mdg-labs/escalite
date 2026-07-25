package graph

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/mdg-labs/escalite/services/api/graph/model"
	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/api/internal/gqlerr"
	"github.com/mdg-labs/escalite/services/api/internal/handlers"
)

func (r *queryResolver) OnCallNow(ctx context.Context, scheduleID string, at *time.Time) (*model.OnCallNow, error) {
	sc, err := requireAdminSession(ctx)
	if err != nil {
		return nil, err
	}

	scheduleUUID, err := parseUUIDField(scheduleID, "scheduleId")
	if err != nil {
		return nil, err
	}

	queries := db.New(r.pool)
	schedule, err := queries.GetScheduleByID(ctx, db.GetScheduleByIDParams{
		ID:             scheduleUUID,
		OrganizationID: sc.User.OrganizationID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		r.logger.Error("load schedule failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	loc, err := time.LoadLocation(schedule.Timezone)
	if err != nil {
		r.logger.Error("load schedule timezone failed", "error", err, "timezone", schedule.Timezone)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	evalAt := time.Now().UTC()
	if at != nil {
		evalAt = at.UTC()
	}

	rotations, err := queries.ListRotationsByScheduleID(ctx, db.ListRotationsByScheduleIDParams{
		ScheduleID:     scheduleUUID,
		OrganizationID: sc.User.OrganizationID,
	})
	if err != nil {
		r.logger.Error("list rotations failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	layers := make([]*model.OnCallLayer, 0, len(rotations))
	for _, rotation := range rotations {
		participantIDs, err := decodeParticipantIDs(rotation.Participants)
		if err != nil {
			r.logger.Error("decode rotation participants failed", "error", err, "rotationId", rotation.ID)
			return nil, gqlerr.New(handlers.CodeInternal, "internal error")
		}

		userID, err := currentOnCallUser(
			rotation.Rrule,
			timeFromDB(rotation.CreatedAt),
			loc,
			evalAt,
			participantIDs,
		)
		if err != nil {
			if errors.Is(err, errNoActiveShift) {
				continue
			}
			r.logger.Error("compute on-call user failed", "error", err, "rotationId", rotation.ID)
			return nil, gqlerr.New(handlers.CodeInternal, "internal error")
		}

		layers = append(layers, &model.OnCallLayer{
			Layer:      int(rotation.Layer),
			RotationID: rotation.ID.String(),
			UserID:     userID,
		})
	}

	return &model.OnCallNow{
		ScheduleID: schedule.ID.String(),
		ComputedAt: evalAt,
		Layers:     layers,
	}, nil
}
