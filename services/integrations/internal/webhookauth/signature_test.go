package webhookauth_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mdg-labs/escalite/services/integrations"
	"github.com/mdg-labs/escalite/services/integrations/internal/webhookauth"
)

func TestVerifyAcceptsHMACSignature(t *testing.T) {
	secret := "top-secret"
	body := []byte(`{"monitor_id":123}`)
	headers := http.Header{}
	headers.Set("X-Escalite-Signature", webhookauth.SignBody(secret, body))

	err := webhookauth.Verify(secret, body, headers)
	require.NoError(t, err)
}

func TestVerifyAcceptsStaticSecretHeader(t *testing.T) {
	secret := "top-secret"
	body := []byte(`{"monitor_id":123}`)
	headers := http.Header{}
	headers.Set("X-Escalite-Signature", secret)

	err := webhookauth.Verify(secret, body, headers)
	require.NoError(t, err)
}

func TestVerifyRejectsInvalidSignature(t *testing.T) {
	body := []byte(`{"monitor_id":123}`)
	headers := http.Header{}
	headers.Set("X-Escalite-Signature", "invalid")

	err := webhookauth.Verify("top-secret", body, headers)
	require.Error(t, err)

	var sigErr integrations.ErrInvalidSignature
	require.ErrorAs(t, err, &sigErr)
}

func TestVerifySkipsWhenSecretUnset(t *testing.T) {
	err := webhookauth.Verify("", []byte(`{}`), http.Header{})
	require.NoError(t, err)
}
