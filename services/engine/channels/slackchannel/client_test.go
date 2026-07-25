package slackchannel_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mdg-labs/escalite/services/engine/channels/slackchannel"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) Do(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestCreateChannelCallsConversationsCreate(t *testing.T) {
	var received createConversationRequest
	slackchannel.SetHTTPClient(roundTripFunc(func(req *http.Request) (*http.Response, error) {
		require.Equal(t, http.MethodPost, req.Method)
		require.Equal(t, "https://slack.com/api/conversations.create", req.URL.String())
		require.Equal(t, "Bearer xoxb-test-token", req.Header.Get("Authorization"))

		body, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(body, &received))

		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"ok":true,"channel":{"id":"C12345678"}}`)),
			Header:     make(http.Header),
		}, nil
	}))
	t.Cleanup(func() { slackchannel.SetHTTPClient(nil) })

	channelID, err := slackchannel.CreateChannel(context.Background(), "xoxb-test-token", "incident-01934f5a")
	require.NoError(t, err)
	require.Equal(t, "C12345678", channelID)
	require.Equal(t, createConversationRequest{Name: "incident-01934f5a"}, received)
}

func TestCreateChannelFailsOnSlackAPIError(t *testing.T) {
	slackchannel.SetHTTPClient(roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"ok":false,"error":"name_taken"}`)),
			Header:     make(http.Header),
		}, nil
	}))
	t.Cleanup(func() { slackchannel.SetHTTPClient(nil) })

	_, err := slackchannel.CreateChannel(context.Background(), "xoxb-test-token", "incident-01934f5a")
	require.Error(t, err)
	require.Contains(t, err.Error(), "name_taken")
}

type createConversationRequest struct {
	Name string `json:"name"`
}
