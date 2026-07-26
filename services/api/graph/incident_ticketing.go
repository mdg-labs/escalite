package graph

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/mdg-labs/escalite/services/api/internal/crypto"
	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/outboundintegrations"
)

func (r *Resolver) tryCreateIncidentTicket(ctx context.Context, queries *db.Queries, incident db.Incident) {
	if r.secrets == nil {
		return
	}

	settings, err := queries.GetOrganizationTicketingSettings(ctx, incident.OrganizationID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			r.logger.Info("ticketing integration not configured, skipping ticket creation",
				"incident_id", incident.ID,
				"organization_id", incident.OrganizationID,
			)
			return
		}
		r.logger.Error("load ticketing settings for incident failed",
			"error", err,
			"incident_id", incident.ID,
			"organization_id", incident.OrganizationID,
		)
		return
	}

	plugin, err := outboundintegrations.Get(settings.PluginName)
	if err != nil {
		r.logger.Error("unknown ticketing plugin configured",
			"error", err,
			"plugin_name", settings.PluginName,
			"incident_id", incident.ID,
			"organization_id", incident.OrganizationID,
		)
		return
	}

	token, err := r.secrets.Decrypt(crypto.Encrypted{
		KeyID:      settings.EncryptionKeyID,
		Ciphertext: settings.ApiTokenCiphertext,
	})
	if err != nil {
		r.logger.Error("decrypt ticketing api token failed",
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
		r.logger.Error("create external incident ticket failed",
			"error", err,
			"plugin_name", settings.PluginName,
			"incident_id", incident.ID,
			"organization_id", incident.OrganizationID,
		)
		return
	}

	if _, err := queries.UpdateIncidentTicketURL(ctx, db.UpdateIncidentTicketURLParams{
		ID:             incident.ID,
		OrganizationID: incident.OrganizationID,
		TicketUrl:      pgtype.Text{String: ticket.URL, Valid: true},
	}); err != nil {
		r.logger.Error("persist incident ticket url failed",
			"error", err,
			"incident_id", incident.ID,
			"organization_id", incident.OrganizationID,
			"ticket_url", ticket.URL,
		)
	}
}
