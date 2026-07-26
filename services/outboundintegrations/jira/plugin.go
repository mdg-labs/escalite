// Package jira implements the Jira outbound ticketing plugin.
package jira

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

const pluginName = "jira"

const defaultTimeout = 15 * time.Second

// HTTPDoer performs outbound HTTP requests.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

var httpClient HTTPDoer = &http.Client{Timeout: defaultTimeout}

// SetHTTPClient configures the HTTP client used by the Jira plugin.
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
	BaseURL    string `json:"base_url"`
	Email      string `json:"email"`
	ProjectKey string `json:"project_key"`
	IssueType  string `json:"issue_type,omitempty"`
}

type createIssueRequest struct {
	Fields createIssueFields `json:"fields"`
}

type createIssueFields struct {
	Project   map[string]string      `json:"project"`
	Summary   string                 `json:"summary"`
	IssueType map[string]string      `json:"issuetype"`
	Description map[string]any       `json:"description,omitempty"`
}

type createIssueResponse struct {
	Key  string `json:"key"`
	Self string `json:"self"`
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

	issueType := strings.TrimSpace(parsed.IssueType)
	if issueType == "" {
		issueType = "Incident"
	}

	body, err := json.Marshal(createIssueRequest{
		Fields: createIssueFields{
			Project: map[string]string{"key": parsed.ProjectKey},
			Summary: incident.Title,
			IssueType: map[string]string{"name": issueType},
			Description: map[string]any{
				"type":    "doc",
				"version": 1,
				"content": []map[string]any{
					{
						"type": "paragraph",
						"content": []map[string]string{
							{
								"type": "text",
								"text": fmt.Sprintf("Escalite incident %s", incident.ID),
							},
						},
					},
				},
			},
		},
	})
	if err != nil {
		return outboundintegrations.Ticket{}, fmt.Errorf("marshal jira issue: %w", err)
	}

	endpoint := strings.TrimRight(parsed.BaseURL, "/") + "/rest/api/3/issue"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return outboundintegrations.Ticket{}, fmt.Errorf("create jira request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", basicAuth(parsed.Email, apiToken))

	resp, err := httpClient.Do(req)
	if err != nil {
		return outboundintegrations.Ticket{}, fmt.Errorf("create jira issue: %w", err)
	}
	defer resp.Body.Close()

	responseBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		if len(responseBody) == 0 {
			return outboundintegrations.Ticket{}, fmt.Errorf("jira returned status %d", resp.StatusCode)
		}
		return outboundintegrations.Ticket{}, fmt.Errorf("jira returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	var created createIssueResponse
	if err := json.Unmarshal(responseBody, &created); err != nil {
		return outboundintegrations.Ticket{}, fmt.Errorf("decode jira response: %w", err)
	}
	if strings.TrimSpace(created.Key) == "" {
		return outboundintegrations.Ticket{}, errors.New("jira response missing issue key")
	}

	return outboundintegrations.Ticket{
		URL: strings.TrimRight(parsed.BaseURL, "/") + "/browse/" + created.Key,
	}, nil
}

func (p plugin) ValidateConfig(cfg json.RawMessage) error {
	return outboundintegrations.ValidateObjectConfig(cfg, map[string]func(string) error{
		"base_url":    outboundintegrations.RequireHTTPURL,
		"email":       outboundintegrations.RequireNonEmpty,
		"project_key": outboundintegrations.RequireNonEmpty,
	})
}

func (p plugin) ConfigSchema() json.RawMessage {
	return json.RawMessage(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"base_url": {
				"type": "string",
				"format": "uri",
				"title": "Jira base URL"
			},
			"email": {
				"type": "string",
				"format": "email",
				"title": "Jira account email"
			},
			"project_key": {
				"type": "string",
				"minLength": 1,
				"title": "Project key"
			},
			"issue_type": {
				"type": "string",
				"title": "Issue type",
				"default": "Incident"
			}
		},
		"required": ["base_url", "email", "project_key"],
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
		"base_url":    outboundintegrations.RequireHTTPURL,
		"email":       outboundintegrations.RequireNonEmpty,
		"project_key": outboundintegrations.RequireNonEmpty,
	}); err != nil {
		return pluginConfig{}, err
	}
	return cfg, nil
}

func basicAuth(username, password string) string {
	token := username + ":" + password
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(token))
}
