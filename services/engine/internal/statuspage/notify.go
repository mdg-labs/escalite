package statuspage

import (
	"context"
	"fmt"
	"html"
	"strings"

	"github.com/google/uuid"

	"github.com/mdg-labs/escalite/services/engine/internal/db"
	engineemail "github.com/mdg-labs/escalite/services/engine/internal/email"
	"github.com/mdg-labs/escalite/services/engine/statuspageapi"
)

// NotifyConfig controls signed unsubscribe links in subscriber emails.
type NotifyConfig struct {
	SigningKey        []byte
	StatusPagePublicURL string
}

// NotifyParams identifies a status page incident update to email subscribers about.
type NotifyParams struct {
	OrganizationID       uuid.UUID
	StatusPageIncidentID uuid.UUID
	UpdateID             uuid.UUID
}

// NotifySubscribers emails active subscribers for the incident's status page.
// When sender is nil (SMTP unset), the call succeeds without sending mail.
func NotifySubscribers(
	ctx context.Context,
	queries *db.Queries,
	sender engineemail.Sender,
	cfg NotifyConfig,
	params NotifyParams,
) error {
	if sender == nil {
		return nil
	}

	incident, err := queries.GetStatusPageIncidentByID(ctx, db.GetStatusPageIncidentByIDParams{
		ID:             params.StatusPageIncidentID,
		OrganizationID: params.OrganizationID,
	})
	if err != nil {
		return fmt.Errorf("load status page incident: %w", err)
	}

	page, err := queries.GetStatusPageByID(ctx, db.GetStatusPageByIDParams{
		ID:             incident.StatusPageID,
		OrganizationID: params.OrganizationID,
	})
	if err != nil {
		return fmt.Errorf("load status page: %w", err)
	}

	update, err := queries.GetStatusPageIncidentUpdateByID(ctx, db.GetStatusPageIncidentUpdateByIDParams{
		ID:             params.UpdateID,
		OrganizationID: params.OrganizationID,
	})
	if err != nil {
		return fmt.Errorf("load status page incident update: %w", err)
	}

	subscribers, err := queries.ListActiveStatusPageSubscriptions(ctx, db.ListActiveStatusPageSubscriptionsParams{
		StatusPageID:   page.ID,
		OrganizationID: params.OrganizationID,
	})
	if err != nil {
		return fmt.Errorf("list status page subscriptions: %w", err)
	}
	if len(subscribers) == 0 {
		return nil
	}

	subject := fmt.Sprintf("[%s] %s", formatIncidentStatus(update.Status), incident.Title)

	for _, subscriber := range subscribers {
		email := strings.TrimSpace(subscriber.Email)
		if email == "" {
			continue
		}

		unsubscribeURL, err := buildUnsubscribeURL(cfg, subscriber.ID, subscriber.OrganizationID)
		if err != nil {
			return fmt.Errorf("build unsubscribe link for %s: %w", email, err)
		}

		textBody := buildTextBody(page.Title, incident.Title, update.Status, update.Body, unsubscribeURL)
		htmlBody := buildHTMLBody(page.Title, incident.Title, update.Status, update.Body, unsubscribeURL)

		if err := sender.Send(ctx, engineemail.Message{
			To:       email,
			Subject:  subject,
			TextBody: textBody,
			HTMLBody: htmlBody,
		}); err != nil {
			return fmt.Errorf("send status page email to %s: %w", email, err)
		}
	}

	return nil
}

func buildUnsubscribeURL(cfg NotifyConfig, subscriptionID, organizationID uuid.UUID) (string, error) {
	if len(cfg.SigningKey) == 0 {
		return "", fmt.Errorf("signing key is required")
	}

	token, err := statuspageapi.SignUnsubscribeToken(cfg.SigningKey, statuspageapi.UnsubscribeClaims{
		SubscriptionID: subscriptionID,
		OrganizationID: organizationID,
	})
	if err != nil {
		return "", err
	}

	return statuspageapi.BuildUnsubscribeURL(cfg.StatusPagePublicURL, token), nil
}

func formatIncidentStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "identified":
		return "Identified"
	case "monitoring":
		return "Monitoring"
	case "resolved":
		return "Resolved"
	default:
		return "Investigating"
	}
}

func buildTextBody(pageTitle, incidentTitle, status, body, unsubscribeURL string) string {
	var text strings.Builder
	text.WriteString(pageTitle)
	text.WriteString("\n\n")
	text.WriteString(incidentTitle)
	text.WriteString("\nStatus: ")
	text.WriteString(formatIncidentStatus(status))
	text.WriteString("\n\n")
	text.WriteString(body)
	text.WriteString("\n\n")
	text.WriteString("Unsubscribe from incident emails:\n")
	text.WriteString(unsubscribeURL)
	return text.String()
}

func buildHTMLBody(pageTitle, incidentTitle, status, body, unsubscribeURL string) string {
	var htmlBody strings.Builder
	htmlBody.WriteString("<!DOCTYPE html><html><body>")
	htmlBody.WriteString("<p><strong>")
	htmlBody.WriteString(html.EscapeString(pageTitle))
	htmlBody.WriteString("</strong></p>")
	htmlBody.WriteString("<h2>")
	htmlBody.WriteString(html.EscapeString(incidentTitle))
	htmlBody.WriteString("</h2>")
	htmlBody.WriteString("<p><strong>Status:</strong> ")
	htmlBody.WriteString(html.EscapeString(formatIncidentStatus(status)))
	htmlBody.WriteString("</p>")
	htmlBody.WriteString("<p>")
	htmlBody.WriteString(html.EscapeString(body))
	htmlBody.WriteString("</p>")
	htmlBody.WriteString("<p><a href=\"")
	htmlBody.WriteString(html.EscapeString(unsubscribeURL))
	htmlBody.WriteString("\">Unsubscribe from incident emails</a></p>")
	htmlBody.WriteString("</body></html>")
	return htmlBody.String()
}
