package alerts

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/api/internal/escalation"
	"github.com/mdg-labs/escalite/services/integrations"
)

// ProcessInbound applies a normalized inbound alert event for an integration key.
// For triggered events it returns the created alert ID; resolved events return a zero UUID.
func ProcessInbound(
	ctx context.Context,
	queries *db.Queries,
	logger *slog.Logger,
	key db.IntegrationKey,
	event integrations.AlertCreate,
) (uuid.UUID, error) {
	switch event.EventType {
	case integrations.EventTriggered:
		return createTriggered(ctx, queries, key, event)
	case integrations.EventResolved:
		return uuid.Nil, resolveByDedupKey(ctx, queries, logger, key, event.DedupKey)
	default:
		return uuid.Nil, nil
	}
}

func createTriggered(ctx context.Context, queries *db.Queries, key db.IntegrationKey, event integrations.AlertCreate) (uuid.UUID, error) {
	_, err := queries.GetOpenAlertByServiceDedupKey(ctx, db.GetOpenAlertByServiceDedupKeyParams{
		ServiceID: key.ServiceID,
		DedupKey:  event.DedupKey,
	})
	if err == nil {
		updated, err := queries.IncrementOpenAlertEventCount(ctx, db.IncrementOpenAlertEventCountParams{
			ServiceID: key.ServiceID,
			DedupKey:  event.DedupKey,
		})
		if err != nil {
			return uuid.Nil, err
		}
		return updated.ID, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, err
	}

	description := pgtype.Text{}
	if event.Description != "" {
		description = pgtype.Text{String: event.Description, Valid: true}
	}

	alert, err := queries.CreateTriggeredAlert(ctx, db.CreateTriggeredAlertParams{
		ID:               uuid.Must(uuid.NewV7()),
		OrganizationID:   key.OrganizationID,
		ServiceID:        key.ServiceID,
		IntegrationKeyID: pgtype.UUID{Bytes: key.ID, Valid: true},
		DedupKey:         event.DedupKey,
		Summary:          event.Summary,
		Description:      description,
		Priority:         normalizePriority(event.Priority),
		EscalationState:  []byte(`{}`),
	})
	if err != nil {
		return uuid.Nil, err
	}
	return alert.ID, nil
}

func resolveByDedupKey(
	ctx context.Context,
	queries *db.Queries,
	logger *slog.Logger,
	key db.IntegrationKey,
	dedupKey string,
) error {
	alert, err := queries.GetOpenAlertByServiceDedupKey(ctx, db.GetOpenAlertByServiceDedupKeyParams{
		ServiceID: key.ServiceID,
		DedupKey:  dedupKey,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			logger.Debug("inbound resolve ignored; no open alert",
				"service_id", key.ServiceID,
				"dedup_key", dedupKey,
			)
			return nil
		}
		return err
	}

	state, err := escalation.ParseState(alert.EscalationState)
	if err != nil {
		return err
	}
	state = escalation.ClearedTimerState(state)

	raw, err := escalation.MarshalState(state)
	if err != nil {
		return err
	}

	_, err = queries.CloseAlert(ctx, db.CloseAlertParams{
		ID:              alert.ID,
		OrganizationID:  key.OrganizationID,
		EscalationState: raw,
	})
	if err != nil && errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	return err
}

func normalizePriority(priority string) string {
	switch priority {
	case "low", "high":
		return priority
	default:
		return "high"
	}
}
