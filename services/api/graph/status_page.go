package graph

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/mdg-labs/escalite/services/api/graph/model"
	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/api/internal/gqlerr"
	"github.com/mdg-labs/escalite/services/api/internal/handlers"
)

var statusPageSlugPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$`)

func statusPageComponentStatusToDB(status model.StatusPageComponentStatus) string {
	switch status {
	case model.StatusPageComponentStatusDegraded:
		return "degraded"
	case model.StatusPageComponentStatusPartialOutage:
		return "partial_outage"
	case model.StatusPageComponentStatusMajorOutage:
		return "major_outage"
	default:
		return "operational"
	}
}

func statusPageComponentStatusFromDB(status string) model.StatusPageComponentStatus {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "degraded":
		return model.StatusPageComponentStatusDegraded
	case "partial_outage":
		return model.StatusPageComponentStatusPartialOutage
	case "major_outage":
		return model.StatusPageComponentStatusMajorOutage
	default:
		return model.StatusPageComponentStatusOperational
	}
}

func statusPageFromDB(page db.StatusPage) *model.StatusPage {
	var frameAncestors *string
	if page.FrameAncestorsCsp.Valid && strings.TrimSpace(page.FrameAncestorsCsp.String) != "" {
		value := page.FrameAncestorsCsp.String
		frameAncestors = &value
	}

	return &model.StatusPage{
		ID:                page.ID.String(),
		OrganizationID:    page.OrganizationID.String(),
		Slug:              page.Slug,
		Title:             page.Title,
		Enabled:           page.Enabled,
		FrameAncestorsCsp: frameAncestors,
		CreatedAt:         timeFromDB(page.CreatedAt),
		UpdatedAt:         timeFromDB(page.UpdatedAt),
	}
}

func statusPageComponentFromDB(component db.StatusPageComponent) *model.StatusPageComponent {
	var description *string
	if component.Description.Valid && strings.TrimSpace(component.Description.String) != "" {
		value := component.Description.String
		description = &value
	}

	var serviceID *string
	if component.ServiceID.Valid {
		value := uuid.UUID(component.ServiceID.Bytes).String()
		serviceID = &value
	}

	return &model.StatusPageComponent{
		ID:           component.ID.String(),
		StatusPageID: component.StatusPageID.String(),
		Name:         component.Name,
		Description:  description,
		Status:       statusPageComponentStatusFromDB(component.Status),
		Position:     int(component.Position),
		ServiceID:    serviceID,
		CreatedAt:    timeFromDB(component.CreatedAt),
		UpdatedAt:    timeFromDB(component.UpdatedAt),
	}
}

func statusPageComponentsFromDB(components []db.StatusPageComponent) []*model.StatusPageComponent {
	result := make([]*model.StatusPageComponent, 0, len(components))
	for _, component := range components {
		result = append(result, statusPageComponentFromDB(component))
	}
	return result
}

func statusPageSubscriptionFromDB(sub db.StatusPageSubscription) *model.StatusPageSubscription {
	return &model.StatusPageSubscription{
		ID:        sub.ID.String(),
		Email:     sub.Email,
		CreatedAt: timeFromDB(sub.CreatedAt),
	}
}

func statusPageSubscriptionsFromDB(subs []db.StatusPageSubscription) []*model.StatusPageSubscription {
	result := make([]*model.StatusPageSubscription, 0, len(subs))
	for _, sub := range subs {
		result = append(result, statusPageSubscriptionFromDB(sub))
	}
	return result
}

func statusPageIncidentUpdateFromDB(update db.StatusPageIncidentUpdate) *model.StatusPageIncidentUpdate {
	return &model.StatusPageIncidentUpdate{
		ID:        update.ID.String(),
		Body:      update.Body,
		Status:    incidentStatusFromDB(update.Status),
		CreatedAt: timeFromDB(update.CreatedAt),
	}
}

func statusPageIncidentUpdatesFromDB(updates []db.StatusPageIncidentUpdate) []*model.StatusPageIncidentUpdate {
	result := make([]*model.StatusPageIncidentUpdate, 0, len(updates))
	for _, update := range updates {
		result = append(result, statusPageIncidentUpdateFromDB(update))
	}
	return result
}

func statusPageIncidentFromDB(incident db.StatusPageIncident, affectedComponentIDs []string, updates []*model.StatusPageIncidentUpdate) *model.StatusPageIncident {
	var resolvedAt = optionalTimeFromDB(incident.ResolvedAt)
	var incidentID *string
	if incident.IncidentID.Valid {
		value := uuid.UUID(incident.IncidentID.Bytes).String()
		incidentID = &value
	}

	return &model.StatusPageIncident{
		ID:                   incident.ID.String(),
		StatusPageID:         incident.StatusPageID.String(),
		IncidentID:           incidentID,
		Title:                incident.Title,
		Status:               incidentStatusFromDB(incident.Status),
		AffectedComponentIds: affectedComponentIDs,
		Updates:              updates,
		ResolvedAt:           resolvedAt,
		CreatedAt:            timeFromDB(incident.CreatedAt),
		UpdatedAt:            timeFromDB(incident.UpdatedAt),
	}
}

func optionalTimeFromDB(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	t := value.Time.UTC()
	return &t
}

func validateStatusPageSlug(slug string) error {
	normalized := strings.ToLower(strings.TrimSpace(slug))
	if normalized == "" {
		return gqlerr.New(handlers.CodeValidation, "slug is required")
	}
	if !statusPageSlugPattern.MatchString(normalized) {
		return gqlerr.New(handlers.CodeValidation, "slug must be lowercase alphanumeric with optional hyphens")
	}
	return nil
}

func (r *Resolver) loadStatusPageForOrg(ctx context.Context, queries *db.Queries, orgID uuid.UUID) (*model.StatusPage, error) {
	page, err := queries.GetStatusPageByOrganizationID(ctx, orgID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		r.logger.Error("load status page failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}
	return r.populateStatusPage(ctx, queries, page)
}

func (r *Resolver) populateStatusPage(ctx context.Context, queries *db.Queries, page db.StatusPage) (*model.StatusPage, error) {
	result := statusPageFromDB(page)

	components, err := queries.ListStatusPageComponents(ctx, db.ListStatusPageComponentsParams{
		StatusPageID:   page.ID,
		OrganizationID: page.OrganizationID,
	})
	if err != nil {
		r.logger.Error("list status page components failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}
	result.Components = statusPageComponentsFromDB(components)

	incidents, err := queries.ListActiveStatusPageIncidents(ctx, db.ListActiveStatusPageIncidentsParams{
		StatusPageID:   page.ID,
		OrganizationID: page.OrganizationID,
	})
	if err != nil {
		r.logger.Error("list status page incidents failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}
	result.Incidents, err = r.statusPageIncidentsFromDB(ctx, queries, page.OrganizationID, incidents)
	if err != nil {
		return nil, err
	}

	subs, err := queries.ListStatusPageSubscriptions(ctx, db.ListStatusPageSubscriptionsParams{
		StatusPageID:   page.ID,
		OrganizationID: page.OrganizationID,
	})
	if err != nil {
		r.logger.Error("list status page subscriptions failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}
	result.Subscriptions = statusPageSubscriptionsFromDB(subs)

	return result, nil
}

func (r *Resolver) statusPageIncidentsFromDB(ctx context.Context, queries *db.Queries, orgID uuid.UUID, incidents []db.StatusPageIncident) ([]*model.StatusPageIncident, error) {
	result := make([]*model.StatusPageIncident, 0, len(incidents))
	for _, incident := range incidents {
		componentIDs, err := queries.ListStatusPageIncidentComponentIDs(ctx, db.ListStatusPageIncidentComponentIDsParams{
			StatusPageIncidentID: incident.ID,
			OrganizationID:       orgID,
		})
		if err != nil {
			r.logger.Error("list status page incident components failed", "error", err)
			return nil, gqlerr.New(handlers.CodeInternal, "internal error")
		}

		affected := make([]string, 0, len(componentIDs))
		for _, id := range componentIDs {
			affected = append(affected, id.String())
		}

		updates, err := queries.ListStatusPageIncidentUpdates(ctx, db.ListStatusPageIncidentUpdatesParams{
			StatusPageIncidentID: incident.ID,
			OrganizationID:       orgID,
		})
		if err != nil {
			r.logger.Error("list status page incident updates failed", "error", err)
			return nil, gqlerr.New(handlers.CodeInternal, "internal error")
		}

		result = append(result, statusPageIncidentFromDB(incident, affected, statusPageIncidentUpdatesFromDB(updates)))
	}
	return result, nil
}

func (r *Resolver) requireOrgStatusPage(ctx context.Context, queries *db.Queries, orgID uuid.UUID) (db.StatusPage, error) {
	page, err := queries.GetStatusPageByOrganizationID(ctx, orgID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.StatusPage{}, gqlerr.New(handlers.CodeValidation, "status page must be configured first")
		}
		r.logger.Error("load status page failed", "error", err)
		return db.StatusPage{}, gqlerr.New(handlers.CodeInternal, "internal error")
	}
	return page, nil
}

func (r *Resolver) setStatusPageIncidentComponents(ctx context.Context, queries *db.Queries, orgID, incidentID uuid.UUID, componentIDs []uuid.UUID) error {
	if err := queries.ReplaceStatusPageIncidentComponents(ctx, db.ReplaceStatusPageIncidentComponentsParams{
		StatusPageIncidentID: incidentID,
		OrganizationID:       orgID,
	}); err != nil {
		r.logger.Error("replace status page incident components failed", "error", err)
		return gqlerr.New(handlers.CodeInternal, "internal error")
	}

	for _, componentID := range componentIDs {
		if err := queries.InsertStatusPageIncidentComponent(ctx, db.InsertStatusPageIncidentComponentParams{
			StatusPageIncidentID:   incidentID,
			StatusPageComponentID: componentID,
			OrganizationID:         orgID,
		}); err != nil {
			r.logger.Error("insert status page incident component failed", "error", err)
			return gqlerr.New(handlers.CodeInternal, "internal error")
		}
	}
	return nil
}

func isStatusPageSlugConflict(err error) bool {
	return isUniqueViolation(err) && strings.Contains(err.Error(), "status_pages_slug_key")
}
