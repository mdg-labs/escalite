package push_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mdg-labs/escalite/services/engine/channels"
	pushchannel "github.com/mdg-labs/escalite/services/engine/channels/push"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) Do(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestSendPostsExpoPushMessage(t *testing.T) {
	var received expoMessage
	pushchannel.SetHTTPClient(roundTripFunc(func(req *http.Request) (*http.Response, error) {
		require.Equal(t, http.MethodPost, req.Method)
		require.Equal(t, "application/json", req.Header.Get("Content-Type"))

		body, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(body, &received))

		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"data":[{"status":"ok","id":"ticket-1"}]}`)),
			Header:     make(http.Header),
		}, nil
	}))
	t.Cleanup(func() { pushchannel.SetHTTPClient(nil) })

	config, err := json.Marshal(map[string]string{
		"expo_push_token": "ExponentPushToken[abc123]",
	})
	require.NoError(t, err)

	channel := pushchannel.New()
	err = channel.Send(context.Background(), channels.SendParams{
		Config: config,
		Alert: channels.Alert{
			ID:          "alert-1",
			ServiceID:   "service-1",
			ServiceName: "checkout-api",
			Status:      "triggered",
			Priority:    "high",
			Summary:     "Disk full",
			Description: "Volume /data is full",
		},
	})
	require.NoError(t, err)
	require.Equal(t, expoMessage{
		To:       "ExponentPushToken[abc123]",
		Title:    "Disk full",
		Body:     "Volume /data is full",
		Priority: "high",
		Data: alertData{
			Type:      "alert.triggered",
			AlertID:   "alert-1",
			ServiceID: "service-1",
			Priority:  "high",
			Title:     "Disk full",
			Body:      "Volume /data is full",
			Actions:   []string{"ack", "escalate"},
		},
	}, received)

	raw, err := json.Marshal(received.Data)
	require.NoError(t, err)
	require.NotContains(t, string(raw), "critical")
}

func TestSendFailsWithoutExpoPushToken(t *testing.T) {
	channel := pushchannel.New()
	err := channel.Send(context.Background(), channels.SendParams{
		Alert: channels.Alert{
			ID:      "alert-1",
			Summary: "Disk full",
			Status:  "triggered",
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "expo_push_token is required")
}

func TestSendFailsOnInvalidExpoToken(t *testing.T) {
	pushchannel.SetHTTPClient(roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(`{
				"data": [{
					"status": "error",
					"message": "\"ExponentPushToken[invalid]\" is not a registered push notification recipient",
					"details": { "error": "DeviceNotRegistered" }
				}]
			}`)),
			Header: make(http.Header),
		}, nil
	}))
	t.Cleanup(func() { pushchannel.SetHTTPClient(nil) })

	config, err := json.Marshal(map[string]string{
		"expo_push_token": "ExponentPushToken[invalid]",
	})
	require.NoError(t, err)

	channel := pushchannel.New()
	err = channel.Send(context.Background(), channels.SendParams{
		Config: config,
		Alert: channels.Alert{
			ID:      "alert-1",
			Summary: "Disk full",
			Status:  "triggered",
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "DeviceNotRegistered")
}

func TestValidateConfigRequiresExpoPushToken(t *testing.T) {
	channel := pushchannel.New()
	err := channel.ValidateConfig(json.RawMessage(`{"expo_push_token":""}`))
	require.Error(t, err)

	err = channel.ValidateConfig(json.RawMessage(`{"expo_push_token":"ExponentPushToken[abc]"}`))
	require.NoError(t, err)
}

type expoMessage struct {
	To       string    `json:"to"`
	Title    string    `json:"title"`
	Body     string    `json:"body"`
	Priority string    `json:"priority,omitempty"`
	Data     alertData `json:"data"`
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
