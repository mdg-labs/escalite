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

func (r *mutationResolver) PromoteAlertToIncident(ctx context.Context, input model.PromoteAlertToIncidentInput) (*model.Alert, error) {
	sc, err := requireAuthSession(ctx)
	if err != nil {
		return nil, err
	}

	alertID, err := parseUUIDField(input.AlertID, "alertId")
	if err != nil {
		return nil, err
	}

	queries := db.New(r.pool)
	alert, service, err := r.loadAlertWithTeamAccess(ctx, queries, sc, alertID)
	if err != nil {
		return nil, err
	}

	if alert.IncidentID.Valid {
		return nil, gqlerr.New(handlers.CodeValidation, "alert is already attached to an incident")
	}
	if alert.Status == "closed" {
		return nil, gqlerr.New(handlers.CodeValidation, "cannot promote a closed alert")
	}

	if input.IncidentID != nil && strings.TrimSpace(*input.IncidentID) != "" {
		return r.attachAlertToExistingIncident(ctx, queries, sc, alertID, service.TeamID, strings.TrimSpace(*input.IncidentID))
	}

	return r.promoteAlertToNewIncident(ctx, queries, sc, alert, alertID, service.TeamID, input.Title)
}

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
