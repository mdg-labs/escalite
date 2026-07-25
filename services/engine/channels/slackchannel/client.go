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
	defaultTimeout         = 10 * time.Second
	conversationsCreateURL = "https://slack.com/api/conversations.create"
	chatPostMessageURL     = "https://slack.com/api/chat.postMessage"
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

type postMessageRequest struct {
	Channel  string `json:"channel"`
	Text     string `json:"text"`
	ThreadTS string `json:"thread_ts,omitempty"`
}

type postMessageResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error"`
	TS    string `json:"ts"`
}

// PostMessage posts a message to a Slack channel and returns the message timestamp.
func PostMessage(ctx context.Context, botToken, channelID, text string) (string, error) {
	return postMessage(ctx, botToken, channelID, text, "")
}

// PostThreadReply posts a threaded reply in a Slack channel.
func PostThreadReply(ctx context.Context, botToken, channelID, threadTS, text string) error {
	_, err := postMessage(ctx, botToken, channelID, text, threadTS)
	return err
}

func postMessage(ctx context.Context, botToken, channelID, text, threadTS string) (string, error) {
	botToken = strings.TrimSpace(botToken)
	if botToken == "" {
		return "", errors.New("slack bot token is not configured")
	}
	channelID = strings.TrimSpace(channelID)
	if channelID == "" {
		return "", errors.New("slack channel id is required")
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return "", errors.New("slack message text is empty")
	}

	body, err := json.Marshal(postMessageRequest{
		Channel:  channelID,
		Text:     text,
		ThreadTS: strings.TrimSpace(threadTS),
	})
	if err != nil {
		return "", fmt.Errorf("marshal slack payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, chatPostMessageURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create slack request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Authorization", "Bearer "+botToken)

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("post slack message: %w", err)
	}
	defer resp.Body.Close()

	responseBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("slack api returned status %d", resp.StatusCode)
	}

	var slackResp postMessageResponse
	if err := json.Unmarshal(responseBody, &slackResp); err != nil {
		return "", fmt.Errorf("parse slack response: %w", err)
	}
	if !slackResp.OK {
		if slackResp.Error == "" {
			return "", errors.New("slack api returned ok=false")
		}
		return "", fmt.Errorf("slack api error: %s", slackResp.Error)
	}
	if strings.TrimSpace(slackResp.TS) == "" {
		return "", errors.New("slack api returned empty message ts")
	}

	return slackResp.TS, nil
}
