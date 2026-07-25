package slackdm_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mdg-labs/escalite/services/engine/channels"
	slackdmchannel "github.com/mdg-labs/escalite/services/engine/channels/slackdm"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) Do(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestSendPostsDirectMessage(t *testing.T) {
	var received postMessageRequest
	slackdmchannel.SetHTTPClient(roundTripFunc(func(req *http.Request) (*http.Response, error) {
		require.Equal(t, http.MethodPost, req.Method)
		require.Equal(t, "Bearer xoxb-test-token", req.Header.Get("Authorization"))

		body, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(body, &received))

		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
			Header:     make(http.Header),
		}, nil
	}))
	t.Cleanup(func() { slackdmchannel.SetHTTPClient(nil) })

	config, err := json.Marshal(map[string]string{
		"slack_user_id": "U12345678",
		"bot_token":     "xoxb-test-token",
	})
	require.NoError(t, err)

	channel := slackdmchannel.New()
	err = channel.Send(context.Background(), channels.SendParams{
		Config: config,
		Alert: channels.Alert{
			ID:          "alert-1",
			ServiceName: "checkout-api",
			Status:      "triggered",
			Priority:    "high",
			Summary:     "Disk full",
			Description: "Volume /data is full",
		},
	})
	require.NoError(t, err)
	require.Equal(t, postMessageRequest{
		Channel: "U12345678",
		Text:    "*[HIGH] Disk full*\nService: checkout-api | Status: triggered\nVolume /data is full\nAlert ID: alert-1",
	}, received)
}

func TestSendFailsWithoutSlackUserID(t *testing.T) {
	channel := slackdmchannel.New()
	err := channel.Send(context.Background(), channels.SendParams{
		Config: json.RawMessage(`{"bot_token":"xoxb-test-token"}`),
		Alert: channels.Alert{
			ID:      "alert-1",
			Summary: "Disk full",
			Status:  "triggered",
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "slack_user_id is required")
}

func TestSendFailsOnSlackAPIError(t *testing.T) {
	slackdmchannel.SetHTTPClient(roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"ok":false,"error":"invalid_auth"}`)),
			Header:     make(http.Header),
		}, nil
	}))
	t.Cleanup(func() { slackdmchannel.SetHTTPClient(nil) })

	config, err := json.Marshal(map[string]string{
		"slack_user_id": "U12345678",
		"bot_token":     "xoxb-test-token",
	})
	require.NoError(t, err)

	channel := slackdmchannel.New()
	err = channel.Send(context.Background(), channels.SendParams{
		Config: config,
		Alert: channels.Alert{
			ID:      "alert-1",
			Summary: "Disk full",
			Status:  "triggered",
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid_auth")
}

func TestValidateConfigRequiresSlackUserID(t *testing.T) {
	channel := slackdmchannel.New()
	err := channel.ValidateConfig(json.RawMessage(`{"slack_user_id":""}`))
	require.Error(t, err)

	err = channel.ValidateConfig(json.RawMessage(`{"slack_user_id":"U12345678"}`))
	require.NoError(t, err)
}

type postMessageRequest struct {
	Channel string `json:"channel"`
	Text    string `json:"text"`
}
