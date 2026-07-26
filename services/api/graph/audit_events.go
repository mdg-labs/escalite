package graph

import (
	"context"
	"encoding/json"
	"errors"
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

const (
	defaultAuditEventsLimit = 50
	maxAuditEventsLimit     = 200
)

type auditEventFilter struct {
	action   pgtype.Text
	fromTime pgtype.Timestamptz
	toTime   pgtype.Timestamptz
	limit    int32
	offset   int32
}

func parseAuditEventFilter(action *string, from, to *time.Time, limit, offset *int) (auditEventFilter, error) {
	filter := auditEventFilter{
		limit:  defaultAuditEventsLimit,
		offset: 0,
	}

	if action != nil {
		trimmed := strings.TrimSpace(*action)
		if trimmed != "" {
			filter.action = pgtype.Text{String: trimmed, Valid: true}
		}
	}

	if from != nil {
		filter.fromTime = pgtype.Timestamptz{Time: from.UTC(), Valid: true}
	}

	if to != nil {
		filter.toTime = pgtype.Timestamptz{Time: to.UTC(), Valid: true}
	}

	if limit != nil {
		if *limit <= 0 {
			return auditEventFilter{}, gqlerr.New(handlers.CodeValidation, "limit must be positive")
		}
		if *limit > maxAuditEventsLimit {
			filter.limit = maxAuditEventsLimit
		} else {
			filter.limit = int32(*limit)
		}
	}

	if offset != nil {
		if *offset < 0 {
			return auditEventFilter{}, gqlerr.New(handlers.CodeValidation, "offset must be non-negative")
		}
		filter.offset = int32(*offset)
	}

	return filter, nil
}

func listAuditEventsFiltered(
	ctx context.Context,
	queries *db.Queries,
	orgID uuid.UUID,
	filter auditEventFilter,
) (*model.AuditEventConnection, error) {
	totalCount, err := queries.CountAuditEventsFiltered(ctx, db.CountAuditEventsFilteredParams{
		OrganizationID: orgID,
		ActionFilter:   filter.action,
		FromTime:       filter.fromTime,
		ToTime:         filter.toTime,
	})
	if err != nil {
		return nil, err
	}

	events, err := queries.ListAuditEventsFiltered(ctx, db.ListAuditEventsFilteredParams{
		OrganizationID: orgID,
		ActionFilter:   filter.action,
		FromTime:       filter.fromTime,
		ToTime:         filter.toTime,
		PageLimit:      filter.limit,
		PageOffset:     filter.offset,
	})
	if err != nil {
		return nil, err
	}

	actorEmails, err := loadAuditActorEmails(ctx, queries, orgID, events)
	if err != nil {
		return nil, err
	}

	return &model.AuditEventConnection{
		Items:      auditEventsFromDB(events, actorEmails),
		TotalCount: int(totalCount),
	}, nil
}

func loadAuditActorEmails(
	ctx context.Context,
	queries *db.Queries,
	orgID uuid.UUID,
	events []db.AuditEvent,
) (map[uuid.UUID]string, error) {
	emails := make(map[uuid.UUID]string)
	for _, event := range events {
		if !event.ActorID.Valid {
			continue
		}
		actorID := uuid.UUID(event.ActorID.Bytes)
		if _, ok := emails[actorID]; ok {
			continue
		}

		user, err := queries.GetUserByID(ctx, db.GetUserByIDParams{
			ID:             actorID,
			OrganizationID: orgID,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				continue
			}
			return nil, err
		}
		emails[actorID] = user.Email
	}
	return emails, nil
}

func auditEventsFromDB(events []db.AuditEvent, actorEmails map[uuid.UUID]string) []*model.AuditEvent {
	items := make([]*model.AuditEvent, 0, len(events))
	for _, event := range events {
		items = append(items, auditEventFromDB(event, actorEmails))
	}
	return items
}

func auditEventFromDB(event db.AuditEvent, actorEmails map[uuid.UUID]string) *model.AuditEvent {
	var actorID *string
	var actorEmail *string
	if event.ActorID.Valid {
		id := uuid.UUID(event.ActorID.Bytes).String()
		actorID = &id
		if email, ok := actorEmails[uuid.UUID(event.ActorID.Bytes)]; ok {
			actorEmail = &email
		}
	}

	var targetType *string
	if event.TargetType.Valid {
		value := event.TargetType.String
		targetType = &value
	}

	var targetID *string
	if event.TargetID.Valid {
		value := uuid.UUID(event.TargetID.Bytes).String()
		targetID = &value
	}

	metadata := map[string]any{}
	if len(event.Metadata) > 0 {
		_ = json.Unmarshal(event.Metadata, &metadata)
	}

	return &model.AuditEvent{
		ID:             event.ID.String(),
		OrganizationID: event.OrganizationID.String(),
		ActorID:        actorID,
		ActorEmail:     actorEmail,
		Action:         event.Action,
		TargetType:     targetType,
		TargetID:       targetID,
		Metadata:       metadata,
		CreatedAt:      timeFromDB(event.CreatedAt),
	}
}
