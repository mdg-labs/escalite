package slackdm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mdg-labs/escalite/services/engine/channels"
)

const (
	name           = "slack-dm"
	defaultTimeout = 10 * time.Second
	slackAPIURL    = "https://slack.com/api/chat.postMessage"
)

// HTTPDoer performs outbound HTTP requests.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

var httpClient HTTPDoer = &http.Client{Timeout: defaultTimeout}

// SetHTTPClient configures the HTTP client used by the Slack DM channel.
func SetHTTPClient(client HTTPDoer) {
	if client == nil {
		httpClient = &http.Client{Timeout: defaultTimeout}
		return
	}
	httpClient = client
}

type channel struct{}

type channelConfig struct {
	SlackUserID string `json:"slack_user_id"`
	BotToken    string `json:"bot_token"`
}

type postMessageRequest struct {
	Channel string `json:"channel"`
	Text    string `json:"text"`
}

type postMessageResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error"`
}

// New returns the Slack DM notification channel plugin.
func New() channels.NotificationChannel {
	return &channel{}
}

func (c *channel) Name() string {
	return name
}

func (c *channel) Send(ctx context.Context, params channels.SendParams) error {
	cfg, err := parseConfig(params.Config)
	if err != nil {
		return err
	}

	slackUserID := strings.TrimSpace(cfg.SlackUserID)
	if slackUserID == "" {
		return errors.New("slack_user_id is required")
	}

	botToken := strings.TrimSpace(cfg.BotToken)
	if botToken == "" {
		return errors.New("slack bot token is not configured")
	}

	body, err := json.Marshal(postMessageRequest{
		Channel: slackUserID,
		Text:    buildMessage(params.Alert),
	})
	if err != nil {
		return fmt.Errorf("marshal slack payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, slackAPIURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create slack request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Authorization", "Bearer "+botToken)

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("deliver slack dm: %w", err)
	}
	defer resp.Body.Close()

	responseBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("slack api returned status %d", resp.StatusCode)
	}

	var slackResp postMessageResponse
	if err := json.Unmarshal(responseBody, &slackResp); err != nil {
		return fmt.Errorf("parse slack response: %w", err)
	}
	if !slackResp.OK {
		if slackResp.Error == "" {
			return errors.New("slack api returned ok=false")
		}
		return fmt.Errorf("slack api error: %s", slackResp.Error)
	}

	return nil
}

func (c *channel) ValidateConfig(cfg json.RawMessage) error {
	return channels.ValidateObjectConfig(cfg, map[string]func(string) error{
		"slack_user_id": channels.RequireNonEmpty,
	})
}

func (c *channel) ConfigSchema() json.RawMessage {
	return json.RawMessage(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"slack_user_id": {
				"type": "string",
				"minLength": 1,
				"title": "Slack user ID"
			}
		},
		"required": ["slack_user_id"],
		"additionalProperties": false
	}`)
}

func parseConfig(raw json.RawMessage) (channelConfig, error) {
	if len(raw) == 0 {
		return channelConfig{}, nil
	}
	var cfg channelConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return channelConfig{}, errors.New("config must be a JSON object")
	}
	return cfg, nil
}

func buildMessage(alert channels.Alert) string {
	var body strings.Builder
	body.WriteString("*[")
	body.WriteString(strings.ToUpper(alert.Priority))
	body.WriteString("] ")
	body.WriteString(alert.Summary)
	body.WriteString("*\n")
	service := strings.TrimSpace(alert.ServiceName)
	if service == "" {
		service = strings.TrimSpace(alert.ServiceID)
	}
	if service != "" {
		body.WriteString("Service: ")
		body.WriteString(service)
		body.WriteString(" | Status: ")
		body.WriteString(alert.Status)
		body.WriteString("\n")
	}
	if strings.TrimSpace(alert.Description) != "" {
		body.WriteString(alert.Description)
		body.WriteString("\n")
	}
	body.WriteString("Alert ID: ")
	body.WriteString(alert.ID)
	return body.String()
}
