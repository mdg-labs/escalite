package slackchannel_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/mdg-labs/escalite/services/engine/channels/slackchannel"
)

func TestPostMessageCallsChatPostMessage(t *testing.T) {
	var received postMessageRequest
	slackchannel.SetHTTPClient(roundTripFunc(func(req *http.Request) (*http.Response, error) {
		require.Equal(t, http.MethodPost, req.Method)
		require.Equal(t, "https://slack.com/api/chat.postMessage", req.URL.String())
		require.Equal(t, "Bearer xoxb-test-token", req.Header.Get("Authorization"))

		body, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(body, &received))

		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"ok":true,"ts":"1710000000.000100"}`)),
			Header:     make(http.Header),
		}, nil
	}))
	t.Cleanup(func() { slackchannel.SetHTTPClient(nil) })

	ts, err := slackchannel.PostMessage(context.Background(), "xoxb-test-token", "C12345678", "hello incident")
	require.NoError(t, err)
	require.Equal(t, "1710000000.000100", ts)
	require.Equal(t, postMessageRequest{
		Channel: "C12345678",
		Text:    "hello incident",
	}, received)
}

func TestPostThreadReplySetsThreadTS(t *testing.T) {
	var received postMessageRequest
	slackchannel.SetHTTPClient(roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(body, &received))

		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"ok":true,"ts":"1710000000.000200"}`)),
			Header:     make(http.Header),
		}, nil
	}))
	t.Cleanup(func() { slackchannel.SetHTTPClient(nil) })

	err := slackchannel.PostThreadReply(context.Background(), "xoxb-test-token", "C12345678", "1710000000.000100", "thread reply")
	require.NoError(t, err)
	require.Equal(t, postMessageRequest{
		Channel:  "C12345678",
		Text:     "thread reply",
		ThreadTS: "1710000000.000100",
	}, received)
}

func TestFormatResolveSummaryIncludesTimeline(t *testing.T) {
	incidentID := uuid.MustParse("01934f5a-0000-7000-8000-000000000001")
	summary := slackchannel.FormatResolveSummary("Checkout outage", incidentID, []slackchannel.TimelineSummaryLine{
		{
			CreatedAt: time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC),
			EventType: "note",
			Body:      "Rollback complete",
			Actor:     "admin@example.com",
		},
	})

	require.Contains(t, summary, "Incident resolved")
	require.Contains(t, summary, "Checkout outage")
	require.Contains(t, summary, "Rollback complete")
	require.Contains(t, summary, "admin@example.com")
}

type postMessageRequest struct {
	Channel  string `json:"channel"`
	Text     string `json:"text"`
	ThreadTS string `json:"thread_ts,omitempty"`
}
