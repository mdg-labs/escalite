package pagerduty

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const defaultAPIBase = "https://api.pagerduty.com"

// Client reads entities from the PagerDuty REST API.
// The API token is held in memory only for the request lifetime and is never persisted.
type Client struct {
	httpClient *http.Client
	baseURL    string
	token      string
	unsupported []UnsupportedObject
}

// NewClient creates a PagerDuty API client. token must come from env at runtime.
func NewClient(token string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 60 * time.Second}
	}
	base := strings.TrimRight(os.Getenv("PAGERDUTY_API_BASE"), "/")
	if base == "" {
		base = defaultAPIBase
	}
	return &Client{
		httpClient: httpClient,
		baseURL:    base,
		token:      strings.TrimSpace(token),
	}
}

func (c *Client) ListUsers() ([]User, error) {
	items, err := c.paginate("/users")
	if err != nil {
		return nil, err
	}
	all, err := decodeItems[apiUser](items)
	if err != nil {
		return nil, err
	}
	out := make([]User, 0, len(all))
	for _, u := range all {
		out = append(out, User{
			ID:       u.ID,
			Name:     u.Name,
			Email:    u.Email,
			Role:     u.Role,
			JobTitle: u.JobTitle,
		})
	}
	return out, nil
}

func (c *Client) ListSchedules() ([]Schedule, error) {
	items, err := c.paginate("/schedules?include[]=schedule_layers&include[]=users")
	if err != nil {
		return nil, err
	}
	all, err := decodeItems[apiSchedule](items)
	if err != nil {
		return nil, err
	}
	out := make([]Schedule, 0, len(all))
	for _, s := range all {
		schedule := Schedule{
			ID:          s.ID,
			Name:        s.Name,
			Description: s.Description,
			TimeZone:    s.TimeZone,
		}
		for _, layer := range s.ScheduleLayers {
			userIDs := make([]string, 0, len(layer.Users))
			for _, u := range layer.Users {
				if u.User != nil && u.User.ID != "" {
					userIDs = append(userIDs, u.User.ID)
				}
			}
			schedule.Layers = append(schedule.Layers, ScheduleLayer{
				ID:                        layer.ID,
				Name:                      layer.Name,
				RotationTurnLengthSeconds: layer.RotationTurnLengthSeconds,
				Users:                     userIDs,
			})
		}
		out = append(out, schedule)
	}
	return out, nil
}

func (c *Client) ListServices() ([]Service, error) {
	items, err := c.paginate("/services")
	if err != nil {
		return nil, err
	}
	all, err := decodeItems[apiService](items)
	if err != nil {
		return nil, err
	}
	out := make([]Service, 0, len(all))
	for _, s := range all {
		policyID := ""
		if s.EscalationPolicy != nil {
			policyID = s.EscalationPolicy.ID
		}
		out = append(out, Service{
			ID:                 s.ID,
			Name:               s.Name,
			Description:        s.Description,
			EscalationPolicyID: policyID,
		})
	}
	return out, nil
}

func (c *Client) ListEscalationPolicies() ([]EscalationPolicy, error) {
	items, err := c.paginate("/escalation_policies")
	if err != nil {
		return nil, err
	}
	all, err := decodeItems[apiEscalationPolicy](items)
	if err != nil {
		return nil, err
	}
	out := make([]EscalationPolicy, 0, len(all))
	for _, p := range all {
		policy := EscalationPolicy{
			ID:          p.ID,
			Name:        p.Name,
			Description: p.Description,
			NumLoops:    p.NumLoops,
		}
		for _, rule := range p.EscalationRules {
			er := EscalationRule{
				ID:                       rule.ID,
				EscalationDelayInMinutes: rule.EscalationDelayInMinutes,
			}
			for _, target := range rule.Targets {
				er.Targets = append(er.Targets, EscalationTarget{
					ID:   target.ID,
					Type: target.Type,
				})
				if !isSupportedTargetType(target.Type) {
					c.unsupported = append(c.unsupported, UnsupportedObject{
						ObjectType: "escalation_target",
						SourceID:   target.ID,
						Name:       target.Type,
						Reason:     unsupportedTargetReason(target.Type),
					})
				}
			}
			policy.EscalationRules = append(policy.EscalationRules, er)
		}
		out = append(out, policy)
	}
	return out, nil
}

func (c *Client) UnsupportedObjects() []UnsupportedObject {
	return append([]UnsupportedObject(nil), c.unsupported...)
}

func (c *Client) Counts() Counts {
	users, _ := c.ListUsers()
	schedules, _ := c.ListSchedules()
	services, _ := c.ListServices()
	policies, _ := c.ListEscalationPolicies()

	layers := 0
	rules := 0
	for _, s := range schedules {
		layers += len(s.Layers)
	}
	for _, p := range policies {
		rules += len(p.EscalationRules)
	}

	return Counts{
		Users:              len(users),
		Schedules:          len(schedules),
		ScheduleLayers:     layers,
		Services:           len(services),
		EscalationPolicies: len(policies),
		EscalationRules:    rules,
		Unsupported:        len(c.unsupported),
	}
}

func (c *Client) paginate(path string) ([]json.RawMessage, error) {
	if c.token == "" {
		return nil, fmt.Errorf("pagerduty API token is required")
	}

	u, err := url.Parse(c.baseURL + path)
	if err != nil {
		return nil, fmt.Errorf("parse url: %w", err)
	}

	q := u.Query()
	if !strings.Contains(path, "include[]") {
		q.Set("limit", "100")
	}
	u.RawQuery = q.Encode()

	var combined []json.RawMessage
	for {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, u.String(), nil)
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("Authorization", "Token token="+c.token)
		req.Header.Set("Accept", "application/vnd.pagerduty+json;version=2")
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("pagerduty API request: %w", err)
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("read response: %w", err)
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("pagerduty API %s: %s", resp.Status, strings.TrimSpace(string(body)))
		}

		var page paginatedResponse
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, fmt.Errorf("decode response: %w", err)
		}
		combined = append(combined, page.Items...)

		if !page.More {
			break
		}
		q.Set("offset", fmt.Sprintf("%d", page.Offset+page.Limit))
		u.RawQuery = q.Encode()
	}
	return combined, nil
}

func decodeItems[T any](items []json.RawMessage) ([]T, error) {
	out := make([]T, 0, len(items))
	for _, raw := range items {
		var item T
		if err := json.Unmarshal(raw, &item); err != nil {
			return nil, fmt.Errorf("decode item: %w", err)
		}
		out = append(out, item)
	}
	return out, nil
}

type paginatedResponse struct {
	Items  []json.RawMessage `json:"-"`
	More   bool              `json:"more"`
	Offset int               `json:"offset"`
	Limit  int               `json:"limit"`
}

func (p *paginatedResponse) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if err := json.Unmarshal(data, &struct {
		More   *bool `json:"more"`
		Offset *int  `json:"offset"`
		Limit  *int  `json:"limit"`
	}{More: &p.More, Offset: &p.Offset, Limit: &p.Limit}); err != nil {
		return err
	}
	for key, val := range raw {
		if key == "more" || key == "offset" || key == "limit" {
			continue
		}
		var items []json.RawMessage
		if err := json.Unmarshal(val, &items); err != nil {
			continue
		}
		p.Items = items
		break
	}
	return nil
}

func isSupportedTargetType(targetType string) bool {
	switch targetType {
	case "user_reference", "schedule_reference":
		return true
	default:
		return false
	}
}

func unsupportedTargetReason(targetType string) string {
	switch targetType {
	case "escalation_policy_reference":
		return "nested escalation policy references are not supported"
	case "team_reference":
		return "team escalation targets are not supported"
	default:
		return "unsupported escalation target type"
	}
}

// API wire types (subset of PagerDuty schema).

type apiUser struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	JobTitle string `json:"job_title"`
}

type apiSchedule struct {
	ID             string             `json:"id"`
	Name           string             `json:"name"`
	Description    string             `json:"description"`
	TimeZone       string             `json:"time_zone"`
	ScheduleLayers []apiScheduleLayer `json:"schedule_layers"`
}

type apiScheduleLayer struct {
	ID                        string          `json:"id"`
	Name                      string          `json:"name"`
	RotationTurnLengthSeconds int             `json:"rotation_turn_length_seconds"`
	Users                     []apiLayerUser  `json:"users"`
}

type apiLayerUser struct {
	User *struct {
		ID string `json:"id"`
	} `json:"user"`
}

type apiService struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Description      string `json:"description"`
	EscalationPolicy *struct {
		ID string `json:"id"`
	} `json:"escalation_policy"`
}

type apiEscalationPolicy struct {
	ID              string              `json:"id"`
	Name            string              `json:"name"`
	Description     string              `json:"description"`
	NumLoops        int                 `json:"num_loops"`
	EscalationRules []apiEscalationRule `json:"escalation_rules"`
}

type apiEscalationRule struct {
	ID                       string              `json:"id"`
	EscalationDelayInMinutes int                 `json:"escalation_delay_in_minutes"`
	Targets                  []apiEscalationTarget `json:"targets"`
}

type apiEscalationTarget struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}
