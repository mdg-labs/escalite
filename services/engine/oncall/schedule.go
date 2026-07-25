package oncall

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/mdg-labs/escalite/services/engine/internal/db"
)

// UsersAt resolves current on-call user IDs for every active layer on a schedule.
func UsersAt(
	ctx context.Context,
	q db.Querier,
	organizationID uuid.UUID,
	scheduleID uuid.UUID,
	at time.Time,
) ([]uuid.UUID, error) {
	schedule, err := q.GetScheduleByID(ctx, db.GetScheduleByIDParams{
		ID:             scheduleID,
		OrganizationID: organizationID,
	})
	if err != nil {
		return nil, fmt.Errorf("load schedule: %w", err)
	}

	loc, err := time.LoadLocation(schedule.Timezone)
	if err != nil {
		return nil, fmt.Errorf("load timezone %q: %w", schedule.Timezone, err)
	}

	rotations, err := q.ListRotationsByScheduleID(ctx, db.ListRotationsByScheduleIDParams{
		ScheduleID:     scheduleID,
		OrganizationID: organizationID,
	})
	if err != nil {
		return nil, fmt.Errorf("list rotations: %w", err)
	}

	activeOverrides, err := q.ListActiveOverridesByScheduleAt(ctx, db.ListActiveOverridesByScheduleAtParams{
		ScheduleID:     scheduleID,
		OrganizationID: organizationID,
		StartsAt:       pgtype.Timestamptz{Time: at, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("list active overrides: %w", err)
	}
	overrideByRotation := overridesByRotation(activeOverrides)

	seen := make(map[uuid.UUID]struct{})
	userIDs := make([]uuid.UUID, 0, len(rotations))

	for _, rotation := range rotations {
		if override, ok := overrideByRotation[rotation.ID]; ok {
			if _, exists := seen[override.UserID]; !exists {
				seen[override.UserID] = struct{}{}
				userIDs = append(userIDs, override.UserID)
			}
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
			timeFromDB(rotation.CreatedAt),
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
		if _, exists := seen[userID]; !exists {
			seen[userID] = struct{}{}
			userIDs = append(userIDs, userID)
		}
	}

	return userIDs, nil
}

func decodeParticipantIDs(data []byte) ([]string, error) {
	if len(data) == 0 {
		return []string{}, nil
	}

	var ids []string
	if err := json.Unmarshal(data, &ids); err != nil {
		return nil, err
	}
	return ids, nil
}

func timeFromDB(value pgtype.Timestamptz) time.Time {
	if !value.Valid {
		return time.Time{}
	}
	return value.Time.UTC()
}

func overridesByRotation(overrides []db.Override) map[uuid.UUID]db.Override {
	result := make(map[uuid.UUID]db.Override, len(overrides))
	for _, override := range overrides {
		if _, exists := result[override.RotationID]; exists {
			continue
		}
		result[override.RotationID] = override
	}
	return result
}
