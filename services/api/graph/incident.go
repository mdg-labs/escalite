package graph

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/mdg-labs/escalite/services/api/graph/model"
	"github.com/mdg-labs/escalite/services/api/internal/auth"
	"github.com/mdg-labs/escalite/services/api/internal/authz"
	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/api/internal/gqlerr"
	"github.com/mdg-labs/escalite/services/api/internal/handlers"
)

func (r *Resolver) loadIncidentWithTeamAccess(
	ctx context.Context,
	queries *db.Queries,
	sc auth.SessionContext,
	incidentID uuid.UUID,
) (db.Incident, error) {
	incident, err := queries.GetIncidentByID(ctx, db.GetIncidentByIDParams{
		ID:             incidentID,
		OrganizationID: sc.User.OrganizationID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.Incident{}, gqlerr.New(handlers.CodeNotFound, "incident not found")
		}
		r.logger.Error("load incident failed", "error", err)
		return db.Incident{}, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	if _, err := authz.CheckTeamAccess(ctx, queries, sc.User, incident.TeamID); err != nil {
		if errors.Is(err, authz.ErrNotFound) || errors.Is(err, authz.ErrForbidden) {
			return db.Incident{}, gqlerr.New(handlers.CodeForbidden, "access denied")
		}
		r.logger.Error("check team access failed", "error", err)
		return db.Incident{}, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	return incident, nil
}

func (r *mutationResolver) insertTimelineEvent(
	ctx context.Context,
	queries *db.Queries,
	orgID uuid.UUID,
	incidentID uuid.UUID,
	actorID uuid.UUID,
	eventType string,
	body string,
	metadata map[string]any,
) (db.TimelineEvent, error) {
	metaBytes := []byte("{}")
	if metadata != nil {
		encoded, err := json.Marshal(metadata)
		if err != nil {
			r.logger.Error("marshal timeline metadata failed", "error", err)
			return db.TimelineEvent{}, gqlerr.New(handlers.CodeInternal, "internal error")
		}
		metaBytes = encoded
	}

	event, err := queries.CreateTimelineEvent(ctx, db.CreateTimelineEventParams{
		ID:             uuid.Must(uuid.NewV7()),
		IncidentID:     incidentID,
		OrganizationID: orgID,
		ActorID:        actorIDParam(actorID),
		EventType:      eventType,
		Body:           body,
		Metadata:       metaBytes,
	})
	if err != nil {
		r.logger.Error("create timeline event failed", "error", err)
		return db.TimelineEvent{}, gqlerr.New(handlers.CodeInternal, "internal error")
	}
	return event, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func (r *queryResolver) resolveIncident(ctx context.Context, id string) (*model.Incident, error) {
	sc, err := requireAuthSession(ctx)
	if err != nil {
		return nil, err
	}

	incidentID, err := parseUUIDField(id, "id")
	if err != nil {
		return nil, err
	}

	queries := db.New(r.pool)
	incident, err := r.loadIncidentWithTeamAccess(ctx, queries, sc, incidentID)
	if err != nil {
		return nil, err
	}

	return incidentFromDB(incident), nil
}

func (r *queryResolver) resolveIncidents(
	ctx context.Context,
	status *model.IncidentStatus,
	teamID *string,
	limit *int,
) ([]*model.Incident, error) {
	sc, err := requireAuthSession(ctx)
	if err != nil {
		return nil, err
	}

	queries := db.New(r.pool)
	listLimit := incidentsListLimit(limit)
	statusFilter := incidentStatusFilter(status)
	teamFilter := optionalUUIDParam(teamID)

	var rows []db.Incident
	if authz.IsAdmin(sc.User.Role) {
		rows, err = queries.ListIncidentsForOrgAdmin(ctx, db.ListIncidentsForOrgAdminParams{
			OrganizationID: sc.User.OrganizationID,
			Limit:          listLimit,
			StatusFilter:   statusFilter,
			TeamIDFilter:   teamFilter,
		})
	} else {
		rows, err = queries.ListIncidentsForTeamMember(ctx, db.ListIncidentsForTeamMemberParams{
			OrganizationID: sc.User.OrganizationID,
			UserID:         sc.User.ID,
			Limit:          listLimit,
			StatusFilter:   statusFilter,
			TeamIDFilter:   teamFilter,
		})
	}
	if err != nil {
		r.logger.Error("list incidents failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	return incidentsFromDB(rows), nil
}

func validateIncidentTitle(title string) (string, error) {
	trimmed := strings.TrimSpace(title)
	if trimmed == "" {
		return "", gqlerr.New(handlers.CodeValidation, "title is required")
	}
	return trimmed, nil
}

func validateTimelineBody(body string) (string, error) {
	trimmed := strings.TrimSpace(body)
	if trimmed == "" {
		return "", gqlerr.New(handlers.CodeValidation, "body is required")
	}
	return trimmed, nil
}

func validateRoleDefinitionName(name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "", gqlerr.New(handlers.CodeValidation, "name is required")
	}
	return trimmed, nil
}
