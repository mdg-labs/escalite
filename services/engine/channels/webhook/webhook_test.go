package webhook_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	allure "github.com/allure-framework/allure-go/commons/gotest"

	"github.com/allure-framework/allure-go/testify/require"

	"github.com/mdg-labs/escalite/services/engine/channels"
	webhookchannel "github.com/mdg-labs/escalite/services/engine/channels/webhook"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) Do(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestSendPostsAlertPayload(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		var received payload
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(a, http.MethodPost, r.Method)
			require.Equal(a, "application/json", r.Header.Get("Content-Type"))
			require.Empty(a, r.Header.Get("X-Escalite-Signature"))

			body, err := io.ReadAll(r.Body)
			require.NoError(a, err)
			require.NoError(a, json.Unmarshal(body, &received))
			w.WriteHeader(http.StatusOK)
		}))
		a.T().Cleanup(server.Close)

		webhookchannel.SetHTTPClient(server.Client())
		a.T().Cleanup(func() { webhookchannel.SetHTTPClient(nil) })

		channel := webhookchannel.New()
		err := channel.Send(context.Background(), channels.SendParams{
			Target: channels.Target{
				Type: "webhook",
				URL:  server.URL,
			},
			Alert: channels.Alert{
				ID:          "alert-1",
				ServiceName: "checkout-api",
				Status:      "triggered",
				Summary:     "Disk full",
				Description: "Volume /data is 99% full",
			},
		})
		require.NoError(a, err)
		require.Equal(a, payload{
			AlertID: "alert-1",
			Service: "checkout-api",
			Status:  "triggered",
			Title:   "Disk full",
			Body:    "Volume /data is 99% full",
		}, received)
	})
}

func TestSendSignsPayloadWhenSecretConfigured(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		const secret = "top-secret"
		var signature string

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			signature = r.Header.Get("X-Escalite-Signature")
			w.WriteHeader(http.StatusAccepted)
		}))
		a.T().Cleanup(server.Close)

		webhookchannel.SetHTTPClient(server.Client())
		a.T().Cleanup(func() { webhookchannel.SetHTTPClient(nil) })

		config, err := json.Marshal(map[string]string{
			"url":            server.URL,
			"signing_secret": secret,
		})
		require.NoError(a, err)

		channel := webhookchannel.New()
		err = channel.Send(context.Background(), channels.SendParams{
			Config: config,
			Alert: channels.Alert{
				ID:          "alert-1",
				ServiceName: "checkout-api",
				Status:      "triggered",
				Summary:     "Disk full",
			},
		})
		require.NoError(a, err)
		require.True(a, strings.HasPrefix(signature, "sha256="))
	})
}

func TestSendFailsWithoutURL(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		channel := webhookchannel.New()
		err := channel.Send(context.Background(), channels.SendParams{
			Alert: channels.Alert{
				ID:      "alert-1",
				Summary: "Disk full",
				Status:  "triggered",
			},
		})
		require.Error(a, err)
		require.Contains(a, err.Error(), "webhook URL is required")
	})
}

func TestSendFailsOnNonSuccessStatus(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "bad gateway", http.StatusBadGateway)
		}))
		a.T().Cleanup(server.Close)

		webhookchannel.SetHTTPClient(server.Client())
		a.T().Cleanup(func() { webhookchannel.SetHTTPClient(nil) })

		channel := webhookchannel.New()
		err := channel.Send(context.Background(), channels.SendParams{
			Target: channels.Target{URL: server.URL},
			Alert: channels.Alert{
				ID:          "alert-1",
				ServiceName: "checkout-api",
				Summary:     "Disk full",
				Status:      "triggered",
			},
		})
		require.Error(a, err)
		require.Contains(a, err.Error(), "502")
		require.Contains(a, err.Error(), "bad gateway")
	})
}

func TestValidateConfigRequiresURL(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		channel := webhookchannel.New()
		err := channel.ValidateConfig(json.RawMessage(`{"url":"not-a-url"}`))
		require.Error(a, err)

		err = channel.ValidateConfig(json.RawMessage(`{"url":"https://example.com/hooks/escalite"}`))
		require.NoError(a, err)
	})
}

func TestValidateConfigRejectsEmptySigningSecret(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		channel := webhookchannel.New()
		err := channel.ValidateConfig(json.RawMessage(`{
			"url":"https://example.com/hooks/escalite",
			"signing_secret":""
		}`))
		require.Error(a, err)
	})
}

type payload struct {
	AlertID string `json:"alert_id"`
	Service string `json:"service"`
	Status  string `json:"status"`
	Title   string `json:"title"`
	Body    string `json:"body"`
}
