package slackchannel

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
)

const (
	defaultTimeout        = 10 * time.Second
	conversationsCreateURL = "https://slack.com/api/conversations.create"
)

// HTTPDoer performs outbound HTTP requests.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

var httpClient HTTPDoer = &http.Client{Timeout: defaultTimeout}

// SetHTTPClient configures the HTTP client used for Slack API calls.
func SetHTTPClient(client HTTPDoer) {
	if client == nil {
		httpClient = &http.Client{Timeout: defaultTimeout}
		return
	}
	httpClient = client
}

type createConversationRequest struct {
	Name string `json:"name"`
}

type createConversationResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error"`
	Channel struct {
		ID string `json:"id"`
	} `json:"channel"`
}

// CreateChannel creates a public Slack channel and returns its channel ID.
func CreateChannel(ctx context.Context, botToken, name string) (string, error) {
	botToken = strings.TrimSpace(botToken)
	if botToken == "" {
		return "", errors.New("slack bot token is not configured")
	}

	channelName := SanitizeChannelName(name)
	if channelName == "" {
		return "", errors.New("slack channel name is empty")
	}

	body, err := json.Marshal(createConversationRequest{Name: channelName})
	if err != nil {
		return "", fmt.Errorf("marshal slack payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, conversationsCreateURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create slack request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Authorization", "Bearer "+botToken)

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("create slack channel: %w", err)
	}
	defer resp.Body.Close()

	responseBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("slack api returned status %d", resp.StatusCode)
	}

	var slackResp createConversationResponse
	if err := json.Unmarshal(responseBody, &slackResp); err != nil {
		return "", fmt.Errorf("parse slack response: %w", err)
	}
	if !slackResp.OK {
		if slackResp.Error == "" {
			return "", errors.New("slack api returned ok=false")
		}
		return "", fmt.Errorf("slack api error: %s", slackResp.Error)
	}
	if strings.TrimSpace(slackResp.Channel.ID) == "" {
		return "", errors.New("slack api returned empty channel id")
	}

	return slackResp.Channel.ID, nil
}
