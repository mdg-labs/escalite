package push_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	allure "github.com/allure-framework/allure-go/commons/gotest"

	"github.com/allure-framework/allure-go/testify/require"

	"github.com/mdg-labs/escalite/services/engine/channels"
	pushchannel "github.com/mdg-labs/escalite/services/engine/channels/push"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) Do(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestSendPostsExpoPushMessage(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		var received expoMessage
		pushchannel.SetHTTPClient(roundTripFunc(func(req *http.Request) (*http.Response, error) {
			require.Equal(a, http.MethodPost, req.Method)
			require.Equal(a, "application/json", req.Header.Get("Content-Type"))

			body, err := io.ReadAll(req.Body)
			require.NoError(a, err)
			require.NoError(a, json.Unmarshal(body, &received))

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"data":[{"status":"ok","id":"ticket-1"}]}`)),
				Header:     make(http.Header),
			}, nil
		}))
		a.T().Cleanup(func() { pushchannel.SetHTTPClient(nil) })

		config, err := json.Marshal(map[string]string{
			"expo_push_token": "ExponentPushToken[abc123]",
		})
		require.NoError(a, err)

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
		require.NoError(a, err)
		require.Equal(a, expoMessage{
			To:                "ExponentPushToken[abc123]",
			Title:             "Disk full",
			Body:              "Volume /data is full",
			Priority:          "high",
			CategoryID:        "alert.triggered",
			InterruptionLevel: "critical",
			ChannelID:         "alerts-critical",
			Data: alertData{
				Type:      "alert.triggered",
				AlertID:   "alert-1",
				ServiceID: "service-1",
				Priority:  "high",
				Title:     "Disk full",
				Body:      "Volume /data is full",
				Actions:   []string{"ack", "escalate"},
				Critical:  true,
			},
		}, received)

		raw, err := json.Marshal(received.Data)
		require.NoError(a, err)
		require.Contains(a, string(raw), `"critical":true`)
	})
}

func TestSendUsesTimeSensitiveFallbackWhenCriticalDisabled(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		var received expoMessage
		pushchannel.SetHTTPClient(roundTripFunc(func(req *http.Request) (*http.Response, error) {
			body, err := io.ReadAll(req.Body)
			require.NoError(a, err)
			require.NoError(a, json.Unmarshal(body, &received))

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"data":[{"status":"ok","id":"ticket-1"}]}`)),
				Header:     make(http.Header),
			}, nil
		}))
		a.T().Cleanup(func() { pushchannel.SetHTTPClient(nil) })

		config, err := json.Marshal(map[string]any{
			"expo_push_token":  "ExponentPushToken[abc123]",
			"critical_enabled": false,
		})
		require.NoError(a, err)

		channel := pushchannel.New()
		err = channel.Send(context.Background(), channels.SendParams{
			Config: config,
			Alert: channels.Alert{
				ID:          "alert-1",
				ServiceID:   "service-1",
				Priority:    "high",
				Summary:     "Disk full",
				Description: "Volume /data is full",
			},
		})
		require.NoError(a, err)
		require.Equal(a, "time-sensitive", received.InterruptionLevel)
		require.Equal(a, "alerts", received.ChannelID)
		require.False(a, received.Data.Critical)
	})
}

func TestSendFailsWithoutExpoPushToken(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		channel := pushchannel.New()
		err := channel.Send(context.Background(), channels.SendParams{
			Alert: channels.Alert{
				ID:      "alert-1",
				Summary: "Disk full",
				Status:  "triggered",
			},
		})
		require.Error(a, err)
		require.Contains(a, err.Error(), "expo_push_token is required")
	})
}

func TestSendFailsOnInvalidExpoToken(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

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
		a.T().Cleanup(func() { pushchannel.SetHTTPClient(nil) })

		config, err := json.Marshal(map[string]string{
			"expo_push_token": "ExponentPushToken[invalid]",
		})
		require.NoError(a, err)

		channel := pushchannel.New()
		err = channel.Send(context.Background(), channels.SendParams{
			Config: config,
			Alert: channels.Alert{
				ID:      "alert-1",
				Summary: "Disk full",
				Status:  "triggered",
			},
		})
		require.Error(a, err)
		require.Contains(a, err.Error(), "DeviceNotRegistered")
	})
}

func TestValidateConfigRequiresExpoPushToken(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		channel := pushchannel.New()
		err := channel.ValidateConfig(json.RawMessage(`{"expo_push_token":""}`))
		require.Error(a, err)

		err = channel.ValidateConfig(json.RawMessage(`{"expo_push_token":"ExponentPushToken[abc]"}`))
		require.NoError(a, err)
	})
}

type expoMessage struct {
	To                string    `json:"to"`
	Title             string    `json:"title"`
	Body              string    `json:"body"`
	Priority          string    `json:"priority,omitempty"`
	CategoryID        string    `json:"categoryId"`
	InterruptionLevel string    `json:"interruptionLevel,omitempty"`
	ChannelID         string    `json:"channelId,omitempty"`
	Data              alertData `json:"data"`
}

type alertData struct {
	Type      string   `json:"type"`
	AlertID   string   `json:"alertId"`
	ServiceID string   `json:"serviceId"`
	Priority  string   `json:"priority"`
	Title     string   `json:"title"`
	Body      string   `json:"body"`
	Actions   []string `json:"actions"`
	Critical  bool     `json:"critical,omitempty"`
}
