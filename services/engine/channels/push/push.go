package push

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
	name           = "push"
	defaultTimeout = 10 * time.Second
	expoPushAPIURL = "https://exp.host/--/api/v2/push/send"
	alertType      = "alert.triggered"
)

// HTTPDoer performs outbound HTTP requests.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

var httpClient HTTPDoer = &http.Client{Timeout: defaultTimeout}

// SetHTTPClient configures the HTTP client used by the push channel.
func SetHTTPClient(client HTTPDoer) {
	if client == nil {
		httpClient = &http.Client{Timeout: defaultTimeout}
		return
	}
	httpClient = client
}

type channel struct{}

type channelConfig struct {
	ExpoPushToken string `json:"expo_push_token"`
}

type alertData struct {
	Type      string   `json:"type"`
	AlertID   string   `json:"alertId"`
	ServiceID string   `json:"serviceId"`
	Priority  string   `json:"priority"`
	Title     string   `json:"title"`
	Body      string   `json:"body"`
	Actions   []string `json:"actions"`
}

type expoMessage struct {
	To       string    `json:"to"`
	Title    string    `json:"title"`
	Body     string    `json:"body"`
	Priority string    `json:"priority,omitempty"`
	Data     alertData `json:"data"`
}

type pushTicket struct {
	Status  string `json:"status"`
	ID      string `json:"id,omitempty"`
	Message string `json:"message,omitempty"`
	Details struct {
		Error string `json:"error"`
	} `json:"details,omitempty"`
}

type pushResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"errors"`
}

// New returns the push notification channel plugin.
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

	token := strings.TrimSpace(cfg.ExpoPushToken)
	if token == "" {
		return errors.New("expo_push_token is required")
	}

	payload := buildMessage(token, params.Alert)
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal expo payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, expoPushAPIURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create expo request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("deliver push notification: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return fmt.Errorf("read expo response: %w", err)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		if len(responseBody) == 0 {
			return fmt.Errorf("expo push api returned status %d", resp.StatusCode)
		}
		return fmt.Errorf("expo push api returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	var expoResp pushResponse
	if err := json.Unmarshal(responseBody, &expoResp); err != nil {
		return fmt.Errorf("parse expo response: %w", err)
	}
	if len(expoResp.Errors) > 0 {
		first := expoResp.Errors[0]
		if first.Message != "" {
			return fmt.Errorf("expo push api error: %s", first.Message)
		}
		if first.Code != "" {
			return fmt.Errorf("expo push api error: %s", first.Code)
		}
		return errors.New("expo push api request failed")
	}

	tickets, err := parsePushTickets(expoResp.Data)
	if err != nil {
		return fmt.Errorf("parse expo push tickets: %w", err)
	}
	if len(tickets) == 0 {
		return errors.New("expo push api returned no tickets")
	}

	ticket := tickets[0]
	if ticket.Status == "ok" {
		return nil
	}

	if ticket.Details.Error != "" {
		return fmt.Errorf("expo push api error: %s", ticket.Details.Error)
	}
	if ticket.Message != "" {
		return errors.New(ticket.Message)
	}
	return errors.New("expo push api returned ticket status error")
}

func (c *channel) ValidateConfig(cfg json.RawMessage) error {
	return channels.ValidateObjectConfig(cfg, map[string]func(string) error{
		"expo_push_token": channels.RequireNonEmpty,
	})
}

func (c *channel) ConfigSchema() json.RawMessage {
	return json.RawMessage(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"expo_push_token": {
				"type": "string",
				"minLength": 1,
				"title": "Expo push token"
			}
		},
		"required": ["expo_push_token"],
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

func buildMessage(token string, alert channels.Alert) expoMessage {
	title := strings.TrimSpace(alert.Summary)
	body := strings.TrimSpace(alert.Description)

	msg := expoMessage{
		To:    token,
		Title: title,
		Body:  body,
		Data: alertData{
			Type:      alertType,
			AlertID:   alert.ID,
			ServiceID: alert.ServiceID,
			Priority:  alert.Priority,
			Title:     title,
			Body:      body,
			Actions:   []string{"ack", "escalate"},
		},
	}
	if strings.EqualFold(strings.TrimSpace(alert.Priority), "high") {
		msg.Priority = "high"
	}
	return msg
}

func parsePushTickets(raw json.RawMessage) ([]pushTicket, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	var tickets []pushTicket
	if err := json.Unmarshal(raw, &tickets); err == nil {
		return tickets, nil
	}

	var ticket pushTicket
	if err := json.Unmarshal(raw, &ticket); err != nil {
		return nil, err
	}
	return []pushTicket{ticket}, nil
}
