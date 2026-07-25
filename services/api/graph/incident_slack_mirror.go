package graph

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/mdg-labs/escalite/services/api/internal/crypto"
	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/engine/channels/slackchannel"
)

type slackMirrorCredentials struct {
	botToken string
}

func (r *Resolver) loadSlackMirrorCredentials(ctx context.Context, queries *db.Queries, orgID uuid.UUID) (slackMirrorCredentials, bool, error) {
	if r.secrets == nil {
		return slackMirrorCredentials{}, false, nil
	}

	settings, err := queries.GetOrganizationSlackSettings(ctx, orgID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return slackMirrorCredentials{}, false, nil
		}
		return slackMirrorCredentials{}, false, err
	}

	token, err := r.secrets.Decrypt(crypto.Encrypted{
		KeyID:      settings.EncryptionKeyID,
		Ciphertext: settings.BotTokenCiphertext,
	})
	if err != nil {
		return slackMirrorCredentials{}, false, err
	}

	return slackMirrorCredentials{botToken: string(token)}, true, nil
}

func actorLabelFromUser(user db.User) string {
	return strings.TrimSpace(user.Email)
}

func (r *Resolver) scheduleMirrorTimelineEventToSlack(
	orgID uuid.UUID,
	incidentID uuid.UUID,
	actorID uuid.UUID,
	eventType string,
	body string,
	metadata map[string]any,
) {
	switch eventType {
	case "note", "status_changed":
	default:
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		r.tryMirrorTimelineEventToSlack(ctx, orgID, incidentID, actorID, eventType, body, metadata)
	}()
}

func (r *Resolver) tryMirrorTimelineEventToSlack(
	ctx context.Context,
	orgID uuid.UUID,
	incidentID uuid.UUID,
	actorID uuid.UUID,
	eventType string,
	body string,
	metadata map[string]any,
) {
	queries := db.New(r.pool)

	incident, err := queries.GetIncidentByID(ctx, db.GetIncidentByIDParams{
		ID:             incidentID,
		OrganizationID: orgID,
	})
	if err != nil {
		r.logger.Error("load incident for slack timeline mirror failed",
			"error", err,
			"incident_id", incidentID,
			"organization_id", orgID,
		)
		return
	}
	if !incident.SlackChannelID.Valid || !incident.SlackThreadTs.Valid {
		return
	}

	creds, ok, err := r.loadSlackMirrorCredentials(ctx, queries, orgID)
	if err != nil {
		r.logger.Error("load slack credentials for timeline mirror failed",
			"error", err,
			"incident_id", incidentID,
			"organization_id", orgID,
		)
		return
	}
	if !ok {
		return
	}

	actor, err := queries.GetUserByID(ctx, db.GetUserByIDParams{
		ID:             actorID,
		OrganizationID: orgID,
	})
	if err != nil {
		r.logger.Error("load actor for slack timeline mirror failed",
			"error", err,
			"incident_id", incidentID,
			"actor_id", actorID,
		)
		return
	}

	actorLabel := actorLabelFromUser(actor)
	channelID := incident.SlackChannelID.String
	threadTS := incident.SlackThreadTs.String

	var message string
	switch eventType {
	case "note":
		message = slackchannel.FormatNoteReply(actorLabel, body)
	case "status_changed":
		fromStatus, toStatus := statusChangeFromMetadata(metadata)
		message = slackchannel.FormatStatusChangeReply(actorLabel, fromStatus, toStatus, body)
	default:
		return
	}

	if err := slackchannel.PostThreadReply(ctx, creds.botToken, channelID, threadTS, message); err != nil {
		r.logger.Error("mirror timeline event to slack failed",
			"error", err,
			"incident_id", incidentID,
			"organization_id", orgID,
			"event_type", eventType,
		)
	}
}

func (r *Resolver) schedulePostIncidentResolveSummary(incident db.Incident) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		r.tryPostIncidentResolveSummary(ctx, incident)
	}()
}

func (r *Resolver) tryPostIncidentResolveSummary(ctx context.Context, incident db.Incident) {
	if !incident.SlackChannelID.Valid {
		return
	}

	queries := db.New(r.pool)
	creds, ok, err := r.loadSlackMirrorCredentials(ctx, queries, incident.OrganizationID)
	if err != nil {
		r.logger.Error("load slack credentials for resolve summary failed",
			"error", err,
			"incident_id", incident.ID,
			"organization_id", incident.OrganizationID,
		)
		return
	}
	if !ok {
		return
	}

	events, err := queries.ListTimelineEventsByIncidentID(ctx, db.ListTimelineEventsByIncidentIDParams{
		IncidentID:     incident.ID,
		OrganizationID: incident.OrganizationID,
	})
	if err != nil {
		r.logger.Error("load timeline for slack resolve summary failed",
			"error", err,
			"incident_id", incident.ID,
			"organization_id", incident.OrganizationID,
		)
		return
	}

	lines := make([]slackchannel.TimelineSummaryLine, 0, len(events))
	for _, event := range events {
		line := slackchannel.TimelineSummaryLine{
			CreatedAt: event.CreatedAt.Time,
			EventType: event.EventType,
			Body:      event.Body,
		}
		if event.ActorID.Valid {
			actor, actorErr := queries.GetUserByID(ctx, db.GetUserByIDParams{
				ID:             uuid.UUID(event.ActorID.Bytes),
				OrganizationID: incident.OrganizationID,
			})
			if actorErr == nil {
				line.Actor = actorLabelFromUser(actor)
			}
		}
		lines = append(lines, line)
	}

	summary := slackchannel.FormatResolveSummary(incident.Title, incident.ID, lines)
	if _, err := slackchannel.PostMessage(ctx, creds.botToken, incident.SlackChannelID.String, summary); err != nil {
		r.logger.Error("post slack resolve summary failed",
			"error", err,
			"incident_id", incident.ID,
			"organization_id", incident.OrganizationID,
		)
	}
}

func statusChangeFromMetadata(metadata map[string]any) (string, string) {
	if metadata == nil {
		return "", ""
	}
	fromStatus, _ := metadata["from_status"].(string)
	toStatus, _ := metadata["to_status"].(string)
	return fromStatus, toStatus
}
