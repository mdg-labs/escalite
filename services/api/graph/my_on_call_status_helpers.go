package graph

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/mdg-labs/escalite/services/api/internal/auth"
	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/api/internal/gqlerr"
	"github.com/mdg-labs/escalite/services/api/internal/handlers"
	"github.com/mdg-labs/escalite/services/engine/oncall"
)

func viewerAssignmentsForSchedule(
	ctx context.Context,
	queries *db.Queries,
	sc auth.SessionContext,
	row db.ListSchedulesForUserTeamsRow,
	at time.Time,
) ([]oncall.ViewerAssignment, error) {
	schedule := row.Schedule

	rotations, err := queries.ListRotationsByScheduleID(ctx, db.ListRotationsByScheduleIDParams{
		ScheduleID:     schedule.ID,
		OrganizationID: sc.User.OrganizationID,
	})
	if err != nil {
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	activeOverrides, err := queries.ListActiveOverridesByScheduleAt(ctx, db.ListActiveOverridesByScheduleAtParams{
		ScheduleID:     schedule.ID,
		OrganizationID: sc.User.OrganizationID,
		StartsAt:       pgtype.Timestamptz{Time: at, Valid: true},
	})
	if err != nil {
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	rotationInputs := make([]oncall.RotationContext, 0, len(rotations))
	for _, rotation := range rotations {
		rotationInputs = append(rotationInputs, oncall.RotationContext{
			ID:           rotation.ID,
			Layer:        rotation.Layer,
			Rrule:        rotation.Rrule,
			CreatedAt:    timeFromDB(rotation.CreatedAt),
			Participants: rotation.Participants,
		})
	}

	overrideInputs := make([]oncall.ActiveOverride, 0, len(activeOverrides))
	for _, override := range activeOverrides {
		overrideInputs = append(overrideInputs, oncall.ActiveOverride{
			RotationID: override.RotationID,
			UserID:     override.UserID,
			EndsAt:     timeFromDB(override.EndsAt),
		})
	}

	assignments, err := oncall.ViewerAssignmentsAt(
		sc.User.ID,
		oncall.ScheduleContext{
			ScheduleID:   schedule.ID,
			ScheduleName: schedule.Name,
			TeamName:     row.TeamName,
			Timezone:     schedule.Timezone,
		},
		rotationInputs,
		overrideInputs,
		at,
	)
	if err != nil {
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	return assignments, nil
}
