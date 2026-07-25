package graph

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/mdg-labs/escalite/services/api/graph/model"
	"github.com/mdg-labs/escalite/services/api/internal/auth"
	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/api/internal/gqlerr"
	"github.com/mdg-labs/escalite/services/api/internal/handlers"
)

func (r *mutationResolver) promoteAlertToNewIncident(
	ctx context.Context,
	queries *db.Queries,
	sc auth.SessionContext,
	alert db.Alert,
	alertID uuid.UUID,
	teamID uuid.UUID,
	titleInput *string,
) (*model.Alert, error) {
	title := strings.TrimSpace(ptrString(titleInput))
	if title == "" {
		title = alert.Summary
	}
	title, err := validateIncidentTitle(title)
	if err != nil {
		return nil, err
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		r.logger.Error("begin promote transaction failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}
	defer tx.Rollback(ctx)

	txQueries := queries.WithTx(tx)
	incidentID := uuid.Must(uuid.NewV7())

	_, err = txQueries.CreateIncident(ctx, db.CreateIncidentParams{
		ID:              incidentID,
		OrganizationID:  sc.User.OrganizationID,
		TeamID:          teamID,
		Title:           title,
		CreatedByUserID: sc.User.ID,
	})
	if err != nil {
		r.logger.Error("create incident for promote failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	updated, err := txQueries.AssignAlertToIncident(ctx, db.AssignAlertToIncidentParams{
		ID:             alertID,
		OrganizationID: sc.User.OrganizationID,
		IncidentID:     pgtype.UUID{Bytes: incidentID, Valid: true},
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, gqlerr.New(handlers.CodeValidation, "cannot promote alert in its current state")
		}
		r.logger.Error("assign alert to incident failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	if _, err := r.insertTimelineEvent(ctx, txQueries, sc.User.OrganizationID, incidentID, sc.User.ID, "declared", title, map[string]any{
		"alert_id": alertID.String(),
	}); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		r.logger.Error("commit promote transaction failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	incident, err := queries.GetIncidentByID(ctx, db.GetIncidentByIDParams{
		ID:             incidentID,
		OrganizationID: sc.User.OrganizationID,
	})
	if err != nil {
		r.logger.Error("load incident after promote failed", "error", err)
	} else {
		r.tryCreateIncidentSlackChannel(ctx, queries, incident)
	}

	r.audit.IncidentCreated(ctx, queries, sc.User.OrganizationID, sc.User.ID, incidentID)
	return alertFromDB(updated, nil), nil
}

func (r *mutationResolver) attachAlertToExistingIncident(
	ctx context.Context,
	queries *db.Queries,
	sc auth.SessionContext,
	alertID uuid.UUID,
	serviceTeamID uuid.UUID,
	incidentIDRaw string,
) (*model.Alert, error) {
	incidentID, err := parseUUIDField(incidentIDRaw, "incidentId")
	if err != nil {
		return nil, err
	}

	incident, err := r.loadIncidentWithTeamAccess(ctx, queries, sc, incidentID)
	if err != nil {
		return nil, err
	}
	if incident.TeamID != serviceTeamID {
		return nil, gqlerr.New(handlers.CodeValidation, "incident team must match the alert service team")
	}

	updated, err := queries.AssignAlertToIncident(ctx, db.AssignAlertToIncidentParams{
		ID:             alertID,
		OrganizationID: sc.User.OrganizationID,
		IncidentID:     pgtype.UUID{Bytes: incidentID, Valid: true},
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, gqlerr.New(handlers.CodeValidation, "cannot promote alert in its current state")
		}
		r.logger.Error("assign alert to existing incident failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	return alertFromDB(updated, nil), nil
}
