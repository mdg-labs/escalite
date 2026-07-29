package oncall

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ScheduleContext identifies a schedule the viewer may be on call for.
type ScheduleContext struct {
	ScheduleID   uuid.UUID
	ScheduleName string
	TeamName     string
	Timezone     string
}

// RotationContext is the rotation metadata needed to resolve viewer assignments.
type RotationContext struct {
	ID           uuid.UUID
	Layer        int32
	Rrule        string
	CreatedAt    time.Time
	Participants []byte
}

// ActiveOverride is an override active at the evaluation instant.
type ActiveOverride struct {
	RotationID uuid.UUID
	UserID     uuid.UUID
	EndsAt     time.Time
}

// ViewerAssignment is one active on-call layer for the viewer.
type ViewerAssignment struct {
	ScheduleID   uuid.UUID
	ScheduleName string
	TeamName     string
	Layer        int
	Until        time.Time
}

// ViewerAssignmentsAt returns active on-call layers for viewerID on one schedule.
func ViewerAssignmentsAt(
	viewerID uuid.UUID,
	schedule ScheduleContext,
	rotations []RotationContext,
	overrides []ActiveOverride,
	at time.Time,
) ([]ViewerAssignment, error) {
	loc, err := time.LoadLocation(schedule.Timezone)
	if err != nil {
		return nil, fmt.Errorf("load timezone %q: %w", schedule.Timezone, err)
	}

	overrideByRotation := make(map[uuid.UUID]ActiveOverride, len(overrides))
	for _, override := range overrides {
		if _, exists := overrideByRotation[override.RotationID]; exists {
			continue
		}
		overrideByRotation[override.RotationID] = override
	}

	assignments := make([]ViewerAssignment, 0, len(rotations))
	for _, rotation := range rotations {
		if override, ok := overrideByRotation[rotation.ID]; ok {
			if override.UserID != viewerID {
				continue
			}
			assignments = append(assignments, ViewerAssignment{
				ScheduleID:   schedule.ScheduleID,
				ScheduleName: schedule.ScheduleName,
				TeamName:     schedule.TeamName,
				Layer:        int(rotation.Layer),
				Until:        override.EndsAt.UTC(),
			})
			continue
		}

		participantIDs, err := decodeParticipantIDs(rotation.Participants)
		if err != nil {
			return nil, fmt.Errorf("decode rotation %s participants: %w", rotation.ID, err)
		}
		if len(participantIDs) == 0 {
			continue
		}

		userIDText, err := CurrentOnCallUser(
			rotation.Rrule,
			rotation.CreatedAt,
			loc,
			at,
			participantIDs,
		)
		if err != nil {
			if IsNoActiveShift(err) {
				continue
			}
			return nil, fmt.Errorf("compute on-call for rotation %s: %w", rotation.ID, err)
		}

		userID, err := uuid.Parse(userIDText)
		if err != nil {
			return nil, fmt.Errorf("parse on-call user id for rotation %s: %w", rotation.ID, err)
		}
		if userID != viewerID {
			continue
		}

		until, err := CurrentShiftEndAt(rotation.Rrule, rotation.CreatedAt, loc, at)
		if err != nil {
			return nil, fmt.Errorf("compute shift end for rotation %s: %w", rotation.ID, err)
		}

		assignments = append(assignments, ViewerAssignment{
			ScheduleID:   schedule.ScheduleID,
			ScheduleName: schedule.ScheduleName,
			TeamName:     schedule.TeamName,
			Layer:        int(rotation.Layer),
			Until:        until,
		})
	}

	return assignments, nil
}
