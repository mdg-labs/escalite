package graph

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/mdg-labs/escalite/services/api/graph/model"
	"github.com/mdg-labs/escalite/services/api/internal/db"
)

const (
	defaultIncidentsListLimit = 100
	maxIncidentsListLimit     = 200
)

func incidentsListLimit(limit *int) int32 {
	if limit == nil || *limit <= 0 {
		return defaultIncidentsListLimit
	}
	if *limit > maxIncidentsListLimit {
		return maxIncidentsListLimit
	}
	return int32(*limit)
}

func incidentStatusFilter(status *model.IncidentStatus) pgtype.Text {
	if status == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: incidentStatusToDB(*status), Valid: true}
}

func incidentStatusToDB(status model.IncidentStatus) string {
	switch status {
	case model.IncidentStatusIdentified:
		return "identified"
	case model.IncidentStatusMonitoring:
		return "monitoring"
	case model.IncidentStatusResolved:
		return "resolved"
	default:
		return "investigating"
	}
}

func incidentStatusFromDB(status string) model.IncidentStatus {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "identified":
		return model.IncidentStatusIdentified
	case "monitoring":
		return model.IncidentStatusMonitoring
	case "resolved":
		return model.IncidentStatusResolved
	default:
		return model.IncidentStatusInvestigating
	}
}

func timelineEventTypeFromDB(eventType string) model.TimelineEventType {
	switch strings.ToLower(strings.TrimSpace(eventType)) {
	case "declared":
		return model.TimelineEventTypeDeclared
	case "status_changed":
		return model.TimelineEventTypeStatusChanged
	case "role_assigned":
		return model.TimelineEventTypeRoleAssigned
	case "role_unassigned":
		return model.TimelineEventTypeRoleUnassigned
	default:
		return model.TimelineEventTypeNote
	}
}

func incidentFromDB(incident db.Incident) *model.Incident {
	var resolvedAt *time.Time
	if incident.ResolvedAt.Valid {
		t := incident.ResolvedAt.Time.UTC()
		resolvedAt = &t
	}

	var slackChannelID *string
	if incident.SlackChannelID.Valid && incident.SlackChannelID.String != "" {
		value := incident.SlackChannelID.String
		slackChannelID = &value
	}

	var ticketURL *string
	if incident.TicketUrl.Valid && incident.TicketUrl.String != "" {
		value := incident.TicketUrl.String
		ticketURL = &value
	}

	return &model.Incident{
		ID:             incident.ID.String(),
		OrganizationID: incident.OrganizationID.String(),
		TeamID:         incident.TeamID.String(),
		Title:          incident.Title,
		Status:         incidentStatusFromDB(incident.Status),
		ResolvedAt:     resolvedAt,
		SlackChannelID: slackChannelID,
		TicketURL:      ticketURL,
		CreatedAt:      timeFromDB(incident.CreatedAt),
		UpdatedAt:      timeFromDB(incident.UpdatedAt),
	}
}

func incidentsFromDB(rows []db.Incident) []*model.Incident {
	result := make([]*model.Incident, 0, len(rows))
	for _, row := range rows {
		result = append(result, incidentFromDB(row))
	}
	return result
}

func timelineEventFromDB(event db.TimelineEvent) *model.TimelineEvent {
	var metadata map[string]any
	if len(event.Metadata) > 0 {
		_ = json.Unmarshal(event.Metadata, &metadata)
	}
	if metadata == nil {
		metadata = map[string]any{}
	}

	return &model.TimelineEvent{
		ID:             event.ID.String(),
		IncidentID:     event.IncidentID.String(),
		OrganizationID: event.OrganizationID.String(),
		EventType:      timelineEventTypeFromDB(event.EventType),
		Body:           event.Body,
		Metadata:       metadata,
		CreatedAt:      timeFromDB(event.CreatedAt),
	}
}

func timelineEventsFromDB(events []db.TimelineEvent) []*model.TimelineEvent {
	result := make([]*model.TimelineEvent, 0, len(events))
	for _, event := range events {
		result = append(result, timelineEventFromDB(event))
	}
	return result
}

func incidentRoleDefinitionFromDB(def db.IncidentRoleDefinition) *model.IncidentRoleDefinition {
	return &model.IncidentRoleDefinition{
		ID:             def.ID.String(),
		OrganizationID: def.OrganizationID.String(),
		Name:           def.Name,
		SortOrder:      int(def.SortOrder),
		CreatedAt:      timeFromDB(def.CreatedAt),
		UpdatedAt:      timeFromDB(def.UpdatedAt),
	}
}

func incidentRoleDefinitionsFromDB(defs []db.IncidentRoleDefinition) []*model.IncidentRoleDefinition {
	result := make([]*model.IncidentRoleDefinition, 0, len(defs))
	for _, def := range defs {
		result = append(result, incidentRoleDefinitionFromDB(def))
	}
	return result
}

func incidentRoleAssignmentFromDB(assignment db.IncidentRoleAssignment) *model.IncidentRoleAssignment {
	return &model.IncidentRoleAssignment{
		ID:         assignment.ID.String(),
		IncidentID: assignment.IncidentID.String(),
		CreatedAt:  timeFromDB(assignment.CreatedAt),
	}
}

func incidentRoleAssignmentsFromDB(assignments []db.IncidentRoleAssignment) []*model.IncidentRoleAssignment {
	result := make([]*model.IncidentRoleAssignment, 0, len(assignments))
	for _, assignment := range assignments {
		result = append(result, incidentRoleAssignmentFromDB(assignment))
	}
	return result
}

func optionalUUIDParam(id *string) pgtype.UUID {
	if id == nil || strings.TrimSpace(*id) == "" {
		return pgtype.UUID{}
	}
	parsed, err := uuid.Parse(strings.TrimSpace(*id))
	if err != nil {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: parsed, Valid: true}
}

func actorIDParam(userID uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: userID, Valid: true}
}

func ptrString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
