package slackdm_test

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
	slackdmchannel "github.com/mdg-labs/escalite/services/engine/channels/slackdm"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) Do(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestSendPostsDirectMessage(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		var received postMessageRequest
		slackdmchannel.SetHTTPClient(roundTripFunc(func(req *http.Request) (*http.Response, error) {
			require.Equal(a, http.MethodPost, req.Method)
			require.Equal(a, "Bearer xoxb-test-token", req.Header.Get("Authorization"))

			body, err := io.ReadAll(req.Body)
			require.NoError(a, err)
			require.NoError(a, json.Unmarshal(body, &received))

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
				Header:     make(http.Header),
			}, nil
		}))
		a.T().Cleanup(func() { slackdmchannel.SetHTTPClient(nil) })

		config, err := json.Marshal(map[string]string{
			"slack_user_id": "U12345678",
			"bot_token":     "xoxb-test-token",
		})
		require.NoError(a, err)

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
		require.NoError(a, err)
		require.Equal(a, postMessageRequest{
			Channel: "U12345678",
			Text:    "*[HIGH] Disk full*\nService: checkout-api | Status: triggered\nVolume /data is full\nAlert ID: alert-1",
			Blocks: []slackBlock{
				{
					Type: "section",
					Text: &slackTextBlock{
						Type: "mrkdwn",
						Text: "*[HIGH] Disk full*\nService: checkout-api | Status: triggered\nVolume /data is full\nAlert ID: alert-1",
					},
				},
				{
					Type: "actions",
					Elements: []slackElement{
						{
							Type: "button",
							Text: slackPlainText{Type: "plain_text", Text: "Acknowledge"},
							Style:    "primary",
							ActionID: "escalite_ack",
							Value:    `{"alert_id":"alert-1"}`,
						},
						{
							Type: "button",
							Text: slackPlainText{Type: "plain_text", Text: "Escalate"},
							ActionID: "escalite_escalate",
							Value:    `{"alert_id":"alert-1"}`,
						},
					},
				},
			},
		}, received)
	})
}

func TestSendFailsWithoutSlackUserID(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		channel := slackdmchannel.New()
		err := channel.Send(context.Background(), channels.SendParams{
			Config: json.RawMessage(`{"bot_token":"xoxb-test-token"}`),
			Alert: channels.Alert{
				ID:      "alert-1",
				Summary: "Disk full",
				Status:  "triggered",
			},
		})
		require.Error(a, err)
		require.Contains(a, err.Error(), "slack_user_id is required")
	})
}

func TestSendFailsOnSlackAPIError(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		slackdmchannel.SetHTTPClient(roundTripFunc(func(_ *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"ok":false,"error":"invalid_auth"}`)),
				Header:     make(http.Header),
			}, nil
		}))
		a.T().Cleanup(func() { slackdmchannel.SetHTTPClient(nil) })

		config, err := json.Marshal(map[string]string{
			"slack_user_id": "U12345678",
			"bot_token":     "xoxb-test-token",
		})
		require.NoError(a, err)

		channel := slackdmchannel.New()
		err = channel.Send(context.Background(), channels.SendParams{
			Config: config,
			Alert: channels.Alert{
				ID:      "alert-1",
				Summary: "Disk full",
				Status:  "triggered",
			},
		})
		require.Error(a, err)
		require.Contains(a, err.Error(), "invalid_auth")
	})
}

func TestValidateConfigRequiresSlackUserID(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		channel := slackdmchannel.New()
		err := channel.ValidateConfig(json.RawMessage(`{"slack_user_id":""}`))
		require.Error(a, err)

		err = channel.ValidateConfig(json.RawMessage(`{"slack_user_id":"U12345678"}`))
		require.NoError(a, err)
	})
}

type postMessageRequest struct {
	Channel string       `json:"channel"`
	Text    string       `json:"text"`
	Blocks  []slackBlock `json:"blocks,omitempty"`
}

type slackBlock struct {
	Type     string          `json:"type"`
	Text     *slackTextBlock `json:"text,omitempty"`
	Elements []slackElement  `json:"elements,omitempty"`
}

type slackTextBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type slackElement struct {
	Type     string         `json:"type"`
	Text     slackPlainText `json:"text"`
	Style    string         `json:"style,omitempty"`
	ActionID string         `json:"action_id"`
	Value    string         `json:"value"`
}

type slackPlainText struct {
	Type string `json:"type"`
	Text string `json:"text"`
}
