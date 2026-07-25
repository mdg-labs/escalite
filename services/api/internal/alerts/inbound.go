package alerts

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/api/internal/escalation"
	"github.com/mdg-labs/escalite/services/engine/escalationapi"
	"github.com/mdg-labs/escalite/services/integrations"
)

// InboundDeps provides optional runtime services for inbound alert processing.
type InboundDeps struct {
	Pool *pgxpool.Pool
	Jobs escalationapi.JobProducer
}

// ProcessInbound applies a normalized inbound alert event for an integration key.
// For triggered events it returns the created alert ID; resolved events return a zero UUID.
func ProcessInbound(
	ctx context.Context,
	queries *db.Queries,
	logger *slog.Logger,
	key db.IntegrationKey,
	event integrations.AlertCreate,
	deps *InboundDeps,
) (uuid.UUID, error) {
	switch event.EventType {
	case integrations.EventTriggered:
		return createTriggered(ctx, queries, key, event, deps)
	case integrations.EventResolved:
		return uuid.Nil, resolveByDedupKey(ctx, queries, logger, key, event.DedupKey)
	default:
		return uuid.Nil, nil
	}
}

func createTriggered(
	ctx context.Context,
	queries *db.Queries,
	key db.IntegrationKey,
	event integrations.AlertCreate,
	deps *InboundDeps,
) (uuid.UUID, error) {
	service, err := queries.GetServiceByID(ctx, db.GetServiceByIDParams{
		ID:             key.ServiceID,
		OrganizationID: key.OrganizationID,
	})
	if err != nil {
		return uuid.Nil, err
	}

	now := time.Now().UTC()
	suppressed, err := queries.ServiceHasActiveIngestionSuppression(ctx, db.ServiceHasActiveIngestionSuppressionParams{
		ServiceID:      key.ServiceID,
		OrganizationID: key.OrganizationID,
		StartsAt:       pgtype.Timestamptz{Time: now, Valid: true},
	})
	if err != nil {
		return uuid.Nil, err
	}
	if suppressed {
		return uuid.Nil, nil
	}

	dedupWindowSeconds := service.DedupWindowSeconds

	_, err = queries.GetOpenAlertByServiceDedupKey(ctx, db.GetOpenAlertByServiceDedupKeyParams{
		ServiceID:          key.ServiceID,
		DedupKey:           event.DedupKey,
		DedupWindowSeconds: dedupWindowSeconds,
	})
	if err == nil {
		updated, err := queries.IncrementOpenAlertEventCount(ctx, db.IncrementOpenAlertEventCountParams{
			ServiceID:          key.ServiceID,
			DedupKey:           event.DedupKey,
			DedupWindowSeconds: dedupWindowSeconds,
		})
		if err != nil {
			return uuid.Nil, err
		}
		if err := renotifyCollapsedAlert(ctx, deps, updated); err != nil {
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

func renotifyCollapsedAlert(ctx context.Context, deps *InboundDeps, alert db.Alert) error {
	if deps == nil || deps.Pool == nil || deps.Jobs == nil {
		return nil
	}
	return escalationapi.RenotifyCollapsedAlert(ctx, deps.Pool, deps.Jobs, alert.ID, alert.OrganizationID)
}

func resolveByDedupKey(
	ctx context.Context,
	queries *db.Queries,
	logger *slog.Logger,
	key db.IntegrationKey,
	dedupKey string,
) error {
	alert, err := queries.GetOpenAlertByServiceDedupKeyForResolve(ctx, db.GetOpenAlertByServiceDedupKeyForResolveParams{
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

	_, err = queries.ResolveOpenAlert(ctx, db.ResolveOpenAlertParams{
		ID:                  alert.ID,
		OrganizationID:      key.OrganizationID,
		EscalationState:     raw,
		ResolvedIntegration: pgtype.Text{String: key.PluginName, Valid: true},
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
