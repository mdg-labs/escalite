// Package servicenow implements the ServiceNow outbound ticketing plugin.
package servicenow

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mdg-labs/escalite/services/outboundintegrations"
)

const pluginName = "servicenow"

const defaultTimeout = 15 * time.Second

// HTTPDoer performs outbound HTTP requests.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

var httpClient HTTPDoer = &http.Client{Timeout: defaultTimeout}

// SetHTTPClient configures the HTTP client used by the ServiceNow plugin.
func SetHTTPClient(client HTTPDoer) {
	if client == nil {
		httpClient = &http.Client{Timeout: defaultTimeout}
		return
	}
	httpClient = client
}

func init() {
	outboundintegrations.Register(plugin{})
}

type plugin struct{}

func (plugin) Name() string { return pluginName }

type pluginConfig struct {
	InstanceURL string `json:"instance_url"`
	Username    string `json:"username"`
}

type createIncidentRequest struct {
	ShortDescription string `json:"short_description"`
	Description      string `json:"description"`
}

type createIncidentResponse struct {
	Result struct {
		SysID  string `json:"sys_id"`
		Number string `json:"number"`
		Link   string `json:"link"`
	} `json:"result"`
}

func (p plugin) CreateTicket(
	ctx context.Context,
	incident outboundintegrations.Incident,
	cfg json.RawMessage,
	apiToken string,
) (outboundintegrations.Ticket, error) {
	parsed, err := parseConfig(cfg)
	if err != nil {
		return outboundintegrations.Ticket{}, err
	}
	if strings.TrimSpace(apiToken) == "" {
		return outboundintegrations.Ticket{}, errors.New("api token is required")
	}

	body, err := json.Marshal(createIncidentRequest{
		ShortDescription: incident.Title,
		Description:      fmt.Sprintf("Escalite incident %s", incident.ID),
	})
	if err != nil {
		return outboundintegrations.Ticket{}, fmt.Errorf("marshal servicenow incident: %w", err)
	}

	endpoint := strings.TrimRight(parsed.InstanceURL, "/") + "/api/now/table/incident"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return outboundintegrations.Ticket{}, fmt.Errorf("create servicenow request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", basicAuth(parsed.Username, apiToken))

	resp, err := httpClient.Do(req)
	if err != nil {
		return outboundintegrations.Ticket{}, fmt.Errorf("create servicenow incident: %w", err)
	}
	defer resp.Body.Close()

	responseBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		if len(responseBody) == 0 {
			return outboundintegrations.Ticket{}, fmt.Errorf("servicenow returned status %d", resp.StatusCode)
		}
		return outboundintegrations.Ticket{}, fmt.Errorf("servicenow returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	var created createIncidentResponse
	if err := json.Unmarshal(responseBody, &created); err != nil {
		return outboundintegrations.Ticket{}, fmt.Errorf("decode servicenow response: %w", err)
	}
	if strings.TrimSpace(created.Result.SysID) == "" {
		return outboundintegrations.Ticket{}, errors.New("servicenow response missing sys_id")
	}

	ticketURL := strings.TrimSpace(created.Result.Link)
	if ticketURL == "" {
		ticketURL = strings.TrimRight(parsed.InstanceURL, "/") +
			"/nav_to.do?uri=incident.do?sys_id=" + created.Result.SysID
	}

	return outboundintegrations.Ticket{URL: ticketURL}, nil
}

func (p plugin) ValidateConfig(cfg json.RawMessage) error {
	return outboundintegrations.ValidateObjectConfig(cfg, map[string]func(string) error{
		"instance_url": outboundintegrations.RequireHTTPURL,
		"username":     outboundintegrations.RequireNonEmpty,
	})
}

func (p plugin) ConfigSchema() json.RawMessage {
	return json.RawMessage(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"instance_url": {
				"type": "string",
				"format": "uri",
				"title": "ServiceNow instance URL"
			},
			"username": {
				"type": "string",
				"minLength": 1,
				"title": "ServiceNow username"
			}
		},
		"required": ["instance_url", "username"],
		"additionalProperties": false
	}`)
}

func parseConfig(raw json.RawMessage) (pluginConfig, error) {
	if len(raw) == 0 {
		return pluginConfig{}, errors.New("config is required")
	}
	var cfg pluginConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return pluginConfig{}, errors.New("config must be a JSON object")
	}
	if err := outboundintegrations.ValidateObjectConfig(raw, map[string]func(string) error{
		"instance_url": outboundintegrations.RequireHTTPURL,
		"username":     outboundintegrations.RequireNonEmpty,
	}); err != nil {
		return pluginConfig{}, err
	}
	return cfg, nil
}

func basicAuth(username, password string) string {
	token := username + ":" + password
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(token))
}
