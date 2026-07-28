package webhookauth_test

import (
	"net/http"
	"testing"

	allure "github.com/allure-framework/allure-go/commons/gotest"

	"github.com/allure-framework/allure-go/testify/require"

	"github.com/mdg-labs/escalite/services/integrations"
	"github.com/mdg-labs/escalite/services/integrations/internal/webhookauth"
)

func TestVerifyAcceptsHMACSignature(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		secret := "top-secret"
		body := []byte(`{"monitor_id":123}`)
		headers := http.Header{}
		headers.Set("X-Escalite-Signature", webhookauth.SignBody(secret, body))

		err := webhookauth.Verify(secret, body, headers)
		require.NoError(a, err)
	})
}

func TestVerifyAcceptsStaticSecretHeader(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		secret := "top-secret"
		body := []byte(`{"monitor_id":123}`)
		headers := http.Header{}
		headers.Set("X-Escalite-Signature", secret)

		err := webhookauth.Verify(secret, body, headers)
		require.NoError(a, err)
	})
}

func TestVerifyRejectsInvalidSignature(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		body := []byte(`{"monitor_id":123}`)
		headers := http.Header{}
		headers.Set("X-Escalite-Signature", "invalid")

		err := webhookauth.Verify("top-secret", body, headers)
		require.Error(a, err)

		var sigErr integrations.ErrInvalidSignature
		require.ErrorAs(a, err, &sigErr)
	})
}

func TestVerifySkipsWhenSecretUnset(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		err := webhookauth.Verify("", []byte(`{}`), http.Header{})
		require.NoError(a, err)
	})
}
