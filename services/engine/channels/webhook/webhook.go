package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
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
	name           = "webhook"
	defaultTimeout = 10 * time.Second
	signatureHeader = "X-Escalite-Signature"
)

// HTTPDoer performs outbound HTTP requests.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

var httpClient HTTPDoer = &http.Client{Timeout: defaultTimeout}

// SetHTTPClient configures the HTTP client used by the webhook channel.
func SetHTTPClient(client HTTPDoer) {
	if client == nil {
		httpClient = &http.Client{Timeout: defaultTimeout}
		return
	}
	httpClient = client
}

type channel struct{}

type channelConfig struct {
	URL           string `json:"url"`
	SigningSecret string `json:"signing_secret,omitempty"`
}

type payload struct {
	AlertID string `json:"alert_id"`
	Service string `json:"service"`
	Status  string `json:"status"`
	Title   string `json:"title"`
	Body    string `json:"body"`
}

// New returns the outbound webhook notification channel plugin.
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

	targetURL := strings.TrimSpace(params.Target.URL)
	if targetURL == "" {
		targetURL = strings.TrimSpace(cfg.URL)
	}
	if targetURL == "" {
		return errors.New("webhook URL is required")
	}

	body, err := json.Marshal(payload{
		AlertID: params.Alert.ID,
		Service: serviceName(params.Alert),
		Status:  params.Alert.Status,
		Title:   params.Alert.Summary,
		Body:    params.Alert.Description,
	})
	if err != nil {
		return fmt.Errorf("marshal webhook payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Escalite-Webhook/1.0")
	if secret := strings.TrimSpace(cfg.SigningSecret); secret != "" {
		req.Header.Set(signatureHeader, signBody(secret, body))
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("deliver webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		responseBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		if len(responseBody) == 0 {
			return fmt.Errorf("webhook returned status %d", resp.StatusCode)
		}
		return fmt.Errorf("webhook returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	return nil
}

func (c *channel) ValidateConfig(cfg json.RawMessage) error {
	fields := map[string]func(string) error{
		"url": channels.RequireHTTPURL,
	}
	if len(cfg) > 0 {
		var values map[string]json.RawMessage
		if err := json.Unmarshal(cfg, &values); err != nil {
			return errors.New("config must be a JSON object")
		}
		if _, ok := values["signing_secret"]; ok {
			fields["signing_secret"] = channels.RequireNonEmpty
		}
	}
	return channels.ValidateObjectConfig(cfg, fields)
}

func (c *channel) ConfigSchema() json.RawMessage {
	return json.RawMessage(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"url": {
				"type": "string",
				"format": "uri",
				"title": "Webhook URL"
			},
			"signing_secret": {
				"type": "string",
				"minLength": 1,
				"title": "Signing secret"
			}
		},
		"required": ["url"],
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

func serviceName(alert channels.Alert) string {
	if name := strings.TrimSpace(alert.ServiceName); name != "" {
		return name
	}
	return strings.TrimSpace(alert.ServiceID)
}

func signBody(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}
