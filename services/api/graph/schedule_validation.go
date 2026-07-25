package graph

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/teambition/rrule-go"

	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/api/internal/gqlerr"
	"github.com/mdg-labs/escalite/services/api/internal/handlers"
)

func validateTimezone(timezone string) error {
	tz := strings.TrimSpace(timezone)
	if tz == "" {
		return gqlerr.New(handlers.CodeValidation, "timezone is required")
	}
	if _, err := time.LoadLocation(tz); err != nil {
		return gqlerr.New(handlers.CodeValidation, "timezone must be a valid IANA timezone")
	}
	return nil
}

func validateRRule(rruleText string) error {
	text := strings.TrimSpace(rruleText)
	if text == "" {
		return gqlerr.New(handlers.CodeValidation, "rrule is required")
	}
	if _, err := rrule.StrToRRule(text); err != nil {
		return gqlerr.New(handlers.CodeValidation, "rrule is invalid: "+err.Error())
	}
	return nil
}

func validateRotationLayer(layer int) error {
	if layer < 1 {
		return gqlerr.New(handlers.CodeValidation, "layer must be at least 1")
	}
	return nil
}

func parseParticipantIDs(participantIDs []string) ([]uuid.UUID, error) {
	if len(participantIDs) == 0 {
		return nil, gqlerr.New(handlers.CodeValidation, "at least one participant is required")
	}

	ids := make([]uuid.UUID, 0, len(participantIDs))
	seen := make(map[uuid.UUID]struct{}, len(participantIDs))
	for i, raw := range participantIDs {
		id, err := parseUUIDField(raw, "participantIds["+strconv.Itoa(i)+"]")
		if err != nil {
			return nil, err
		}
		if _, ok := seen[id]; ok {
			return nil, gqlerr.New(handlers.CodeValidation, "participantIds must be unique")
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids, nil
}

func encodeParticipantIDs(ids []uuid.UUID) ([]byte, error) {
	strings := make([]string, 0, len(ids))
	for _, id := range ids {
		strings = append(strings, id.String())
	}
	return json.Marshal(strings)
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

func ensureParticipantsExist(ctx context.Context, q db.Querier, orgID uuid.UUID, participantIDs []uuid.UUID) error {
	for _, participantID := range participantIDs {
		if _, err := q.GetUserByID(ctx, db.GetUserByIDParams{
			ID:             participantID,
			OrganizationID: orgID,
		}); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return gqlerr.New(handlers.CodeValidation, "participant user not found")
			}
			return gqlerr.New(handlers.CodeInternal, "internal error")
		}
	}
	return nil
}

func isTimezoneCheckViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23514" && strings.Contains(pgErr.ConstraintName, "timezone")
}

func isRotationLayerConflict(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505" && strings.Contains(pgErr.ConstraintName, "layer")
}
