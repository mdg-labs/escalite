package slackchannel_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	allure "github.com/allure-framework/allure-go/commons/gotest"

	"github.com/allure-framework/allure-go/testify/require"

	"github.com/mdg-labs/escalite/services/engine/channels/slackchannel"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) Do(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestCreateChannelCallsConversationsCreate(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		var received createConversationRequest
		slackchannel.SetHTTPClient(roundTripFunc(func(req *http.Request) (*http.Response, error) {
			require.Equal(a, http.MethodPost, req.Method)
			require.Equal(a, "https://slack.com/api/conversations.create", req.URL.String())
			require.Equal(a, "Bearer xoxb-test-token", req.Header.Get("Authorization"))

			body, err := io.ReadAll(req.Body)
			require.NoError(a, err)
			require.NoError(a, json.Unmarshal(body, &received))

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"ok":true,"channel":{"id":"C12345678"}}`)),
				Header:     make(http.Header),
			}, nil
		}))
		a.T().Cleanup(func() { slackchannel.SetHTTPClient(nil) })

		channelID, err := slackchannel.CreateChannel(context.Background(), "xoxb-test-token", "incident-01934f5a")
		require.NoError(a, err)
		require.Equal(a, "C12345678", channelID)
		require.Equal(a, createConversationRequest{Name: "incident-01934f5a"}, received)
	})
}

func TestCreateChannelFailsOnSlackAPIError(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		slackchannel.SetHTTPClient(roundTripFunc(func(_ *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"ok":false,"error":"name_taken"}`)),
				Header:     make(http.Header),
			}, nil
		}))
		a.T().Cleanup(func() { slackchannel.SetHTTPClient(nil) })

		_, err := slackchannel.CreateChannel(context.Background(), "xoxb-test-token", "incident-01934f5a")
		require.Error(a, err)
		require.Contains(a, err.Error(), "name_taken")
	})
}

type createConversationRequest struct {
	Name string `json:"name"`
}
