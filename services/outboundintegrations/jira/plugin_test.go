package jira_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mdg-labs/escalite/services/outboundintegrations"
	"github.com/mdg-labs/escalite/services/outboundintegrations/jira"
	_ "github.com/mdg-labs/escalite/services/outboundintegrations/jira"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) Do(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestCreateTicketReturnsBrowseURL(t *testing.T) {
	plugin, err := outboundintegrations.Get("jira")
	require.NoError(t, err)

	jira.SetHTTPClient(roundTripFunc(func(req *http.Request) (*http.Response, error) {
		require.Equal(t, http.MethodPost, req.Method)
		require.Equal(t, "https://example.atlassian.net/rest/api/3/issue", req.URL.String())
		require.Equal(t, "Basic "+basicAuth("ops@example.com", "secret-token"), req.Header.Get("Authorization"))

		body, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))

		return &http.Response{
			StatusCode: http.StatusCreated,
			Body:       io.NopCloser(strings.NewReader(`{"id":"10000","key":"OPS-42","self":"https://example.atlassian.net/rest/api/3/issue/10000"}`)),
			Header:     make(http.Header),
		}, nil
	}))
	t.Cleanup(func() { jira.SetHTTPClient(nil) })

	cfg := json.RawMessage(`{
		"base_url": "https://example.atlassian.net",
		"email": "ops@example.com",
		"project_key": "OPS"
	}`)

	ticket, err := plugin.CreateTicket(context.Background(), outboundintegrations.Incident{
		ID:    "01234567-89ab-cdef-0123-456789abcdef",
		Title: "Checkout degradation",
	}, cfg, "secret-token")
	require.NoError(t, err)
	require.Equal(t, "https://example.atlassian.net/browse/OPS-42", ticket.URL)
}

func basicAuth(username, password string) string {
	return base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
}
