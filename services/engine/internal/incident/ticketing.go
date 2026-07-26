package incident

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/mdg-labs/escalite/services/engine/internal/crypto"
	"github.com/mdg-labs/escalite/services/engine/internal/db"
	"github.com/mdg-labs/escalite/services/outboundintegrations"
)

// TicketingConfig configures best-effort external ticket creation for declared incidents.
type TicketingConfig struct {
	Secrets *crypto.Box
	Logger  *slog.Logger
}

var ticketingConfig TicketingConfig

// ConfigureTicketing wires outbound ticketing for auto-promoted incidents.
func ConfigureTicketing(cfg TicketingConfig) {
	ticketingConfig = cfg
}

func maybeCreateIncidentTicket(ctx context.Context, q db.Querier, incident db.Incident) {
	if ticketingConfig.Secrets == nil {
		return
	}

	logger := ticketingConfig.Logger
	if logger == nil {
		logger = slog.Default()
	}

	settings, err := q.GetOrganizationTicketingSettings(ctx, incident.OrganizationID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			logger.Info("ticketing integration not configured, skipping ticket creation",
				"incident_id", incident.ID,
				"organization_id", incident.OrganizationID,
			)
			return
		}
		logger.Error("load ticketing settings for incident failed",
			"error", err,
			"incident_id", incident.ID,
			"organization_id", incident.OrganizationID,
		)
		return
	}

	plugin, err := outboundintegrations.Get(settings.PluginName)
	if err != nil {
		logger.Error("unknown ticketing plugin configured",
			"error", err,
			"plugin_name", settings.PluginName,
			"incident_id", incident.ID,
			"organization_id", incident.OrganizationID,
		)
		return
	}

	token, err := ticketingConfig.Secrets.Decrypt(crypto.Encrypted{
		KeyID:      settings.EncryptionKeyID,
		Ciphertext: settings.ApiTokenCiphertext,
	})
	if err != nil {
		logger.Error("decrypt ticketing api token failed",
			"error", err,
			"incident_id", incident.ID,
			"organization_id", incident.OrganizationID,
		)
		return
	}

	ticket, err := plugin.CreateTicket(ctx, outboundintegrations.Incident{
		ID:             incident.ID.String(),
		OrganizationID: incident.OrganizationID.String(),
		TeamID:         incident.TeamID.String(),
		Title:          incident.Title,
	}, settings.Config, string(token))
	if err != nil {
		logger.Error("create external incident ticket failed",
			"error", err,
			"plugin_name", settings.PluginName,
			"incident_id", incident.ID,
			"organization_id", incident.OrganizationID,
		)
		return
	}

	if _, err := q.UpdateIncidentTicketURL(ctx, db.UpdateIncidentTicketURLParams{
		ID:             incident.ID,
		OrganizationID: incident.OrganizationID,
		TicketUrl:      pgtype.Text{String: ticket.URL, Valid: true},
	}); err != nil {
		logger.Error("persist incident ticket url failed",
			"error", err,
			"incident_id", incident.ID,
			"organization_id", incident.OrganizationID,
			"ticket_url", ticket.URL,
		)
	}
}
