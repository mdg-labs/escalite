package email

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"strings"

	"github.com/mdg-labs/escalite/services/engine/channels"
	engineemail "github.com/mdg-labs/escalite/services/engine/internal/email"
)

const name = "email"

var sender engineemail.Sender

// SetSender configures the SMTP sender used by the email channel.
func SetSender(s engineemail.Sender) {
	sender = s
}

type channel struct{}

// New returns the email notification channel plugin.
func New() channels.NotificationChannel {
	return &channel{}
}

func (c *channel) Name() string {
	return name
}

func (c *channel) Send(ctx context.Context, params channels.SendParams) error {
	if sender == nil {
		return errors.New("smtp not configured: set ESCALITE_SMTP_HOST and ESCALITE_SMTP_FROM (see .env.example)")
	}

	to := strings.TrimSpace(params.Target.Email)
	if to == "" {
		return errors.New("recipient email is required")
	}

	subject := fmt.Sprintf("[%s] %s", strings.ToUpper(params.Alert.Priority), params.Alert.Summary)
	textBody := buildTextBody(params.Alert)
	htmlBody := buildHTMLBody(params.Alert)

	return sender.Send(ctx, engineemail.Message{
		To:       to,
		Subject:  subject,
		TextBody: textBody,
		HTMLBody: htmlBody,
	})
}

func buildTextBody(alert channels.Alert) string {
	var body strings.Builder
	body.WriteString("Alert: ")
	body.WriteString(alert.Summary)
	body.WriteString("\nPriority: ")
	body.WriteString(alert.Priority)
	body.WriteString("\nStatus: ")
	body.WriteString(alert.Status)
	if strings.TrimSpace(alert.Description) != "" {
		body.WriteString("\n\n")
		body.WriteString(alert.Description)
	}
	body.WriteString("\n\nAlert ID: ")
	body.WriteString(alert.ID)
	return body.String()
}

func buildHTMLBody(alert channels.Alert) string {
	var body strings.Builder
	body.WriteString("<!DOCTYPE html><html><body>")
	body.WriteString("<h2>")
	body.WriteString(html.EscapeString(alert.Summary))
	body.WriteString("</h2>")
	body.WriteString("<p><strong>Priority:</strong> ")
	body.WriteString(html.EscapeString(alert.Priority))
	body.WriteString("<br><strong>Status:</strong> ")
	body.WriteString(html.EscapeString(alert.Status))
	body.WriteString("</p>")
	if strings.TrimSpace(alert.Description) != "" {
		body.WriteString("<p>")
		body.WriteString(html.EscapeString(alert.Description))
		body.WriteString("</p>")
	}
	body.WriteString("<p><small>Alert ID: ")
	body.WriteString(html.EscapeString(alert.ID))
	body.WriteString("</small></p>")
	body.WriteString("</body></html>")
	return body.String()
}

func (c *channel) ValidateConfig(cfg json.RawMessage) error {
	if len(cfg) == 0 {
		return nil
	}
	var values map[string]json.RawMessage
	if err := json.Unmarshal(cfg, &values); err != nil {
		return errors.New("config must be a JSON object")
	}
	if len(values) > 0 {
		return errors.New("email channel does not accept config fields")
	}
	return nil
}

func (c *channel) ConfigSchema() json.RawMessage {
	return json.RawMessage(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"additionalProperties": false
	}`)
}
