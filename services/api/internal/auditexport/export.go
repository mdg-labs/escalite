package auditexport

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/mdg-labs/escalite/services/api/internal/db"
)

// Row is one audit event row for CSV export.
type Row struct {
	ID         uuid.UUID
	CreatedAt  time.Time
	Action     string
	ActorID    *uuid.UUID
	ActorEmail string
	TargetType string
	TargetID   *uuid.UUID
	Metadata   string
}

// Export writes audit events as CSV with RFC3339 timestamps.
func Export(rows []Row) ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	if err := writer.Write([]string{
		"id",
		"created_at",
		"action",
		"actor_id",
		"actor_email",
		"target_type",
		"target_id",
		"metadata",
	}); err != nil {
		return nil, fmt.Errorf("write csv header: %w", err)
	}

	for _, row := range rows {
		actorID := ""
		if row.ActorID != nil {
			actorID = row.ActorID.String()
		}
		targetID := ""
		if row.TargetID != nil {
			targetID = row.TargetID.String()
		}

		if err := writer.Write([]string{
			row.ID.String(),
			row.CreatedAt.UTC().Format(time.RFC3339),
			row.Action,
			actorID,
			row.ActorEmail,
			row.TargetType,
			targetID,
			row.Metadata,
		}); err != nil {
			return nil, fmt.Errorf("write csv row: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("flush csv: %w", err)
	}

	return buf.Bytes(), nil
}

// RowsFromDB converts audit events and actor emails into export rows.
func RowsFromDB(events []db.AuditEvent, actorEmails map[uuid.UUID]string) ([]Row, error) {
	rows := make([]Row, 0, len(events))
	for _, event := range events {
		var actorID *uuid.UUID
		actorEmail := ""
		if event.ActorID.Valid {
			id := uuid.UUID(event.ActorID.Bytes)
			actorID = &id
			actorEmail = actorEmails[id]
		}

		targetType := ""
		if event.TargetType.Valid {
			targetType = event.TargetType.String
		}

		var targetID *uuid.UUID
		if event.TargetID.Valid {
			id := uuid.UUID(event.TargetID.Bytes)
			targetID = &id
		}

		metadata := string(event.Metadata)
		if len(event.Metadata) > 0 {
			var compact json.RawMessage
			if err := json.Unmarshal(event.Metadata, &compact); err == nil {
				if encoded, err := json.Marshal(compact); err == nil {
					metadata = string(encoded)
				}
			}
		}
		if metadata == "" {
			metadata = "{}"
		}

		createdAt := time.Time{}
		if event.CreatedAt.Valid {
			createdAt = event.CreatedAt.Time.UTC()
		}

		rows = append(rows, Row{
			ID:         event.ID,
			CreatedAt:  createdAt,
			Action:     event.Action,
			ActorID:    actorID,
			ActorEmail: actorEmail,
			TargetType: targetType,
			TargetID:   targetID,
			Metadata:   metadata,
		})
	}

	return rows, nil
}

// OptionalTime converts a pgtype.Timestamptz to *time.Time.
func OptionalTime(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	t := value.Time.UTC()
	return &t
}
