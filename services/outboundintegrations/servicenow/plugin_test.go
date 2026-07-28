package servicenow_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	allure "github.com/allure-framework/allure-go/commons/gotest"

	"github.com/allure-framework/allure-go/testify/require"

	"github.com/mdg-labs/escalite/services/outboundintegrations"
	"github.com/mdg-labs/escalite/services/outboundintegrations/servicenow"
	_ "github.com/mdg-labs/escalite/services/outboundintegrations/servicenow"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) Do(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestCreateTicketReturnsIncidentURL(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		plugin, err := outboundintegrations.Get("servicenow")
		require.NoError(a, err)

		servicenow.SetHTTPClient(roundTripFunc(func(req *http.Request) (*http.Response, error) {
			require.Equal(a, http.MethodPost, req.Method)
			require.Equal(a, "https://example.service-now.com/api/now/table/incident", req.URL.String())

			return &http.Response{
				StatusCode: http.StatusCreated,
				Body: io.NopCloser(strings.NewReader(`{
					"result": {
						"sys_id": "abc123",
						"number": "INC0010001",
						"link": "https://example.service-now.com/incident.do?sys_id=abc123"
					}
				}`)),
				Header: make(http.Header),
			}, nil
		}))
		a.T().Cleanup(func() { servicenow.SetHTTPClient(nil) })

		cfg := json.RawMessage(`{
			"instance_url": "https://example.service-now.com",
			"username": "integration.user"
		}`)

		ticket, err := plugin.CreateTicket(context.Background(), outboundintegrations.Incident{
			ID:    "01234567-89ab-cdef-0123-456789abcdef",
			Title: "Checkout degradation",
		}, cfg, "secret-password")
		require.NoError(a, err)
		require.Equal(a, "https://example.service-now.com/incident.do?sys_id=abc123", ticket.URL)
	})
}
