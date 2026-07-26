package incident

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/mdg-labs/escalite/services/engine/internal/db"
)

// MaybeAutoPromote evaluates the service auto-promote rule after a new alert is created.
func MaybeAutoPromote(ctx context.Context, q db.Querier, alert db.Alert) error {
	service, err := q.GetServiceByID(ctx, db.GetServiceByIDParams{
		ID:             alert.ServiceID,
		OrganizationID: alert.OrganizationID,
	})
	if err != nil {
		return fmt.Errorf("load service for auto-promote: %w", err)
	}
	if !service.AutoPromoteEnabled {
		return nil
	}

	count, err := q.CountRecentOpenAlertsByService(ctx, db.CountRecentOpenAlertsByServiceParams{
		ServiceID:      alert.ServiceID,
		OrganizationID: alert.OrganizationID,
		WindowSeconds: service.AutoPromoteWindowSeconds,
	})
	if err != nil {
		return fmt.Errorf("count recent alerts: %w", err)
	}
	if count < service.AutoPromoteAlertThreshold {
		return nil
	}

	incident, created, err := findOrCreateOpenIncident(ctx, q, service, alert)
	if err != nil {
		return err
	}

	alerts, err := q.ListRecentUnassignedAlertsByService(ctx, db.ListRecentUnassignedAlertsByServiceParams{
		ServiceID:      alert.ServiceID,
		OrganizationID: alert.OrganizationID,
		WindowSeconds: service.AutoPromoteWindowSeconds,
	})
	if err != nil {
		return fmt.Errorf("list recent unassigned alerts: %w", err)
	}

	for _, candidate := range alerts {
		if _, err := q.AssignAlertToIncident(ctx, db.AssignAlertToIncidentParams{
			ID:             candidate.ID,
			OrganizationID: candidate.OrganizationID,
			IncidentID:     pgtype.UUID{Bytes: incident.ID, Valid: true},
		}); err != nil {
			return fmt.Errorf("assign alert %s to incident: %w", candidate.ID, err)
		}
	}

	if created {
		meta, err := json.Marshal(map[string]any{
			"auto_promote": true,
			"service_id":   service.ID.String(),
			"alert_count":  count,
		})
		if err != nil {
			return fmt.Errorf("marshal auto-promote metadata: %w", err)
		}

		if _, err := q.CreateTimelineEvent(ctx, db.CreateTimelineEventParams{
			ID:             uuid.Must(uuid.NewV7()),
			IncidentID:     incident.ID,
			OrganizationID: service.OrganizationID,
			ActorID:        pgtype.UUID{},
			EventType:      "declared",
			Body:           incident.Title,
			Metadata:       meta,
		}); err != nil {
			return fmt.Errorf("create auto-promote timeline event: %w", err)
		}

		maybeCreateSlackChannel(ctx, q, incident)
		maybeCreateIncidentTicket(ctx, q, incident)
	}

	return nil
}

func findOrCreateOpenIncident(
	ctx context.Context,
	q db.Querier,
	service db.Service,
	triggerAlert db.Alert,
) (db.Incident, bool, error) {
	incident, err := q.GetOpenIncidentForTeamWithServiceAlerts(ctx, db.GetOpenIncidentForTeamWithServiceAlertsParams{
		OrganizationID: service.OrganizationID,
		TeamID:         service.TeamID,
		ServiceID:      service.ID,
	})
	if err == nil {
		return incident, false, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return db.Incident{}, false, fmt.Errorf("find open incident with service alerts: %w", err)
	}

	incident, err = q.GetOpenIncidentForTeam(ctx, db.GetOpenIncidentForTeamParams{
		OrganizationID: service.OrganizationID,
		TeamID:         service.TeamID,
	})
	if err == nil {
		return incident, false, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return db.Incident{}, false, fmt.Errorf("find open incident for team: %w", err)
	}

	admin, err := q.GetFirstAdminUserByOrganization(ctx, service.OrganizationID)
	if err != nil {
		return db.Incident{}, false, fmt.Errorf("load org admin for auto-promote: %w", err)
	}

	title := triggerAlert.Summary
	incidentID := uuid.Must(uuid.NewV7())
	incident, err = q.CreateIncident(ctx, db.CreateIncidentParams{
		ID:              incidentID,
		OrganizationID:  service.OrganizationID,
		TeamID:          service.TeamID,
		Title:           title,
		CreatedByUserID: admin.ID,
	})
	if err != nil {
		return db.Incident{}, false, fmt.Errorf("create auto-promoted incident: %w", err)
	}

	return incident, true, nil
}
