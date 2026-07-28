package twilio_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	allure "github.com/allure-framework/allure-go/commons/gotest"
	"time"

	"github.com/allure-framework/allure-go/testify/assert"
	"github.com/allure-framework/allure-go/testify/require"
	twiliogo "github.com/twilio/twilio-go"
	twilioclient "github.com/twilio/twilio-go/client"

	"github.com/mdg-labs/escalite/services/engine/internal/smsprovider"
	twilioprovider "github.com/mdg-labs/escalite/services/engine/internal/smsprovider/twilio"
)

const (
	testAccountSID = "AC1234567890abcdef1234567890abcd"
	testAuthToken  = "testauthtoken"
	testFromNumber = "+15551234567"
	testToNumber   = "+15559876543"
)

func TestProviderName(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		provider := twilioprovider.New(twilioprovider.Config{
			AccountSID: testAccountSID,
			AuthToken:  testAuthToken,
			FromNumber: testFromNumber,
		})
		assert.Equal(a, smsprovider.ProviderTwilio, provider.Name())
	})
}

func TestSendSMSRequiresConfiguredProvider(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		provider := twilioprovider.New(twilioprovider.Config{})

		err := provider.SendSMS(context.Background(), smsprovider.SMSParams{
			To:      testToNumber,
			Message: "hello",
		})
		require.ErrorIs(a, err, smsprovider.ErrNotConfigured)
	})
}

func TestSendSMSValidatesRecipient(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		provider := twilioprovider.New(twilioprovider.Config{
			AccountSID: testAccountSID,
			AuthToken:  testAuthToken,
			FromNumber: testFromNumber,
		})

		err := provider.SendSMS(context.Background(), smsprovider.SMSParams{
			To:      "5551234567",
			Message: "hello",
		})
		require.Error(a, err)
		assert.Contains(a, err.Error(), "E.164")
	})
}

func TestSendSMSIntegration(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		const message = "Disk full on checkout-api"

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(a, http.MethodPost, r.Method)
			require.Equal(a, fmt.Sprintf("/2010-04-01/Accounts/%s/Messages.json", testAccountSID), r.URL.Path)
			require.Equal(a, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))

			body, err := io.ReadAll(r.Body)
			require.NoError(a, err)
			values, err := url.ParseQuery(string(body))
			require.NoError(a, err)
			require.Equal(a, testToNumber, values.Get("To"))
			require.Equal(a, testFromNumber, values.Get("From"))
			require.Equal(a, message, values.Get("Body"))

			user, pass, ok := r.BasicAuth()
			require.True(a, ok)
			require.Equal(a, testAccountSID, user)
			require.Equal(a, testAuthToken, pass)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"sid":"SM123","status":"queued"}`))
		}))
		a.T().Cleanup(server.Close)

		provider := newTestProvider(t, server.URL)
		err := provider.SendSMS(context.Background(), smsprovider.SMSParams{
			To:      testToNumber,
			Message: message,
		})
		require.NoError(a, err)
	})
}

func TestMakeVoiceCallIntegration(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		const alertTitle = "Disk full"
		const serviceName = "checkout-api"
		const spokenMessage = alertTitle + " on " + serviceName

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(a, http.MethodPost, r.Method)
			require.Equal(a, fmt.Sprintf("/2010-04-01/Accounts/%s/Calls.json", testAccountSID), r.URL.Path)
			require.Equal(a, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))

			body, err := io.ReadAll(r.Body)
			require.NoError(a, err)
			values, err := url.ParseQuery(string(body))
			require.NoError(a, err)
			require.Equal(a, testToNumber, values.Get("To"))
			require.Equal(a, testFromNumber, values.Get("From"))

			twiml := values.Get("Twiml")
			require.Contains(a, twiml, "<Say>")
			require.Contains(a, twiml, alertTitle)
			require.Contains(a, twiml, serviceName)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"sid":"CA123","status":"queued"}`))
		}))
		a.T().Cleanup(server.Close)

		provider := newTestProvider(t, server.URL)
		err := provider.MakeVoiceCall(context.Background(), smsprovider.VoiceParams{
			To:      testToNumber,
			Message: spokenMessage,
		})
		require.NoError(a, err)
	})
}

func newTestProvider(t *testing.T, apiBase string) *twilioprovider.Provider {
	t.Helper()

	client := newTestTwilioClient(apiBase, testAccountSID, testAuthToken)
	return twilioprovider.New(twilioprovider.Config{
		AccountSID: testAccountSID,
		AuthToken:  testAuthToken,
		FromNumber: testFromNumber,
		Client:     client,
	})
}

func newTestTwilioClient(apiBase, accountSID, authToken string) *twiliogo.RestClient {
	inner := &twilioclient.Client{
		Credentials: twilioclient.NewCredentials(accountSID, authToken),
		HTTPClient:  &http.Client{Timeout: 5 * time.Second},
	}
	inner.SetAccountSid(accountSID)

	return twiliogo.NewRestClientWithParams(twiliogo.ClientParams{
		Client: &redirectClient{apiBase: strings.TrimRight(apiBase, "/"), inner: inner},
	})
}

type redirectClient struct {
	apiBase string
	inner   *twilioclient.Client
}

func (c *redirectClient) AccountSid() string {
	return c.inner.AccountSid()
}

func (c *redirectClient) SetTimeout(timeout time.Duration) {
	c.inner.SetTimeout(timeout)
}

func (c *redirectClient) SetOauth(auth twilioclient.OAuth) {
	c.inner.SetOauth(auth)
}

func (c *redirectClient) OAuth() twilioclient.OAuth {
	return c.inner.OAuth()
}

func (c *redirectClient) SendRequest(
	method string,
	rawURL string,
	data url.Values,
	headers map[string]interface{},
	body ...byte,
) (*http.Response, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	base, err := url.Parse(c.apiBase)
	if err != nil {
		return nil, err
	}
	parsed.Scheme = base.Scheme
	parsed.Host = base.Host
	return c.inner.SendRequest(method, parsed.String(), data, headers, body...)
}
