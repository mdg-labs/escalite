package graph

import (
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/engine/oncall"
)

func computeReplacedUserID(timezone string, rotation db.Rotation, at time.Time) pgtype.UUID {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return pgtype.UUID{}
	}

	participantIDs, err := decodeParticipantIDs(rotation.Participants)
	if err != nil || len(participantIDs) == 0 {
		return pgtype.UUID{}
	}

	userID, err := oncall.CurrentOnCallUser(
		rotation.Rrule,
		timeFromDB(rotation.CreatedAt),
		loc,
		at,
		participantIDs,
	)
	if err != nil {
		return pgtype.UUID{}
	}

	replacedID, err := uuid.Parse(userID)
	if err != nil {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: replacedID, Valid: true}
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
