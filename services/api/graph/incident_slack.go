package graph

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/mdg-labs/escalite/services/api/internal/crypto"
	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/engine/channels/slackchannel"
)

func (r *Resolver) tryCreateIncidentSlackChannel(ctx context.Context, queries *db.Queries, incident db.Incident) {
	if r.secrets == nil {
		return
	}

	settings, err := queries.GetOrganizationSlackSettings(ctx, incident.OrganizationID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return
		}
		r.logger.Error("load slack settings for incident channel failed",
			"error", err,
			"incident_id", incident.ID,
			"organization_id", incident.OrganizationID,
		)
		return
	}

	token, err := r.secrets.Decrypt(crypto.Encrypted{
		KeyID:      settings.EncryptionKeyID,
		Ciphertext: settings.BotTokenCiphertext,
	})
	if err != nil {
		r.logger.Error("decrypt slack bot token for incident channel failed",
			"error", err,
			"incident_id", incident.ID,
			"organization_id", incident.OrganizationID,
		)
		return
	}

	template := r.slackIncidentChannelNameTemplate
	if template == "" {
		template = slackchannel.DefaultNameTemplate
	}

	channelName := slackchannel.RenderChannelName(template, incident.ID, incident.Title)
	channelID, err := slackchannel.CreateChannel(ctx, string(token), channelName)
	if err != nil {
		r.logger.Error("create slack incident channel failed",
			"error", err,
			"incident_id", incident.ID,
			"organization_id", incident.OrganizationID,
			"channel_name", channelName,
		)
		return
	}

	if _, err := queries.UpdateIncidentSlackChannelID(ctx, db.UpdateIncidentSlackChannelIDParams{
		ID:             incident.ID,
		OrganizationID: incident.OrganizationID,
		SlackChannelID: pgtype.Text{String: channelID, Valid: true},
	}); err != nil {
		r.logger.Error("persist slack incident channel id failed",
			"error", err,
			"incident_id", incident.ID,
			"organization_id", incident.OrganizationID,
			"slack_channel_id", channelID,
		)
	}
}
