package incident

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/mdg-labs/escalite/services/engine/channels/slackchannel"
	"github.com/mdg-labs/escalite/services/engine/internal/crypto"
	"github.com/mdg-labs/escalite/services/engine/internal/db"
)

// SlackChannelConfig configures best-effort Slack channel creation for declared incidents.
type SlackChannelConfig struct {
	Template string
	Secrets  *crypto.Box
	Logger   *slog.Logger
}

var slackChannelConfig SlackChannelConfig

// ConfigureSlackChannels wires Slack incident channel provisioning for auto-promote.
func ConfigureSlackChannels(cfg SlackChannelConfig) {
	slackChannelConfig = cfg
}

func maybeCreateSlackChannel(ctx context.Context, q db.Querier, incident db.Incident) {
	if slackChannelConfig.Secrets == nil {
		return
	}

	logger := slackChannelConfig.Logger
	if logger == nil {
		logger = slog.Default()
	}

	settings, err := q.GetOrganizationSlackSettings(ctx, incident.OrganizationID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return
		}
		logger.Error("load slack settings for incident channel failed",
			"error", err,
			"incident_id", incident.ID,
			"organization_id", incident.OrganizationID,
		)
		return
	}

	token, err := slackChannelConfig.Secrets.Decrypt(crypto.Encrypted{
		KeyID:      settings.EncryptionKeyID,
		Ciphertext: settings.BotTokenCiphertext,
	})
	if err != nil {
		logger.Error("decrypt slack bot token for incident channel failed",
			"error", err,
			"incident_id", incident.ID,
			"organization_id", incident.OrganizationID,
		)
		return
	}

	template := slackChannelConfig.Template
	if template == "" {
		template = slackchannel.DefaultNameTemplate
	}

	channelName := slackchannel.RenderChannelName(template, incident.ID, incident.Title)
	channelID, err := slackchannel.CreateChannel(ctx, string(token), channelName)
	if err != nil {
		logger.Error("create slack incident channel failed",
			"error", err,
			"incident_id", incident.ID,
			"organization_id", incident.OrganizationID,
			"channel_name", channelName,
		)
		return
	}

	if _, err := q.UpdateIncidentSlackChannelID(ctx, db.UpdateIncidentSlackChannelIDParams{
		ID:             incident.ID,
		OrganizationID: incident.OrganizationID,
		SlackChannelID: pgtype.Text{String: channelID, Valid: true},
	}); err != nil {
		logger.Error("persist slack incident channel id failed",
			"error", err,
			"incident_id", incident.ID,
			"organization_id", incident.OrganizationID,
			"slack_channel_id", channelID,
		)
	}
}
