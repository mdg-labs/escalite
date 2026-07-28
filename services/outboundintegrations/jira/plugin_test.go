package jira_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	allure "github.com/allure-framework/allure-go/commons/gotest"

	"github.com/allure-framework/allure-go/testify/require"

	"github.com/mdg-labs/escalite/services/outboundintegrations"
	"github.com/mdg-labs/escalite/services/outboundintegrations/jira"
	_ "github.com/mdg-labs/escalite/services/outboundintegrations/jira"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) Do(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestCreateTicketReturnsBrowseURL(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		plugin, err := outboundintegrations.Get("jira")
		require.NoError(a, err)

		jira.SetHTTPClient(roundTripFunc(func(req *http.Request) (*http.Response, error) {
			require.Equal(a, http.MethodPost, req.Method)
			require.Equal(a, "https://example.atlassian.net/rest/api/3/issue", req.URL.String())
			require.Equal(a, "Basic "+basicAuth("ops@example.com", "secret-token"), req.Header.Get("Authorization"))

			body, err := io.ReadAll(req.Body)
			require.NoError(a, err)
			var payload map[string]any
			require.NoError(a, json.Unmarshal(body, &payload))

			return &http.Response{
				StatusCode: http.StatusCreated,
				Body:       io.NopCloser(strings.NewReader(`{"id":"10000","key":"OPS-42","self":"https://example.atlassian.net/rest/api/3/issue/10000"}`)),
				Header:     make(http.Header),
			}, nil
		}))
		a.T().Cleanup(func() { jira.SetHTTPClient(nil) })

		cfg := json.RawMessage(`{
			"base_url": "https://example.atlassian.net",
			"email": "ops@example.com",
			"project_key": "OPS"
		}`)

		ticket, err := plugin.CreateTicket(context.Background(), outboundintegrations.Incident{
			ID:    "01234567-89ab-cdef-0123-456789abcdef",
			Title: "Checkout degradation",
		}, cfg, "secret-token")
		require.NoError(a, err)
		require.Equal(a, "https://example.atlassian.net/browse/OPS-42", ticket.URL)
	})
}

func basicAuth(username, password string) string {
	return base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
}
