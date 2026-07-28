package crypto_test

import (
	"encoding/hex"
	"testing"

	allure "github.com/allure-framework/allure-go/commons/gotest"
	"github.com/allure-framework/allure-go/testify/assert"
	"github.com/allure-framework/allure-go/testify/require"

	"github.com/mdg-labs/escalite/services/api/internal/crypto"
)

const testKeyHex = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func testKey(t *testing.T) []byte {
	t.Helper()

	key, err := hex.DecodeString(testKeyHex)
	require.NoError(t, err)
	return key
}

func TestNewBoxRejectsInvalidKeyLength(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		_, err := crypto.NewBox([]byte("too-short"))
		require.Error(a, err)
		assert.Contains(a, err.Error(), "32 bytes")
	})
}

func TestSecretHint(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		assert.Equal(a, "mnop", crypto.SecretHint("abcdefghijklmnop"))
		assert.Equal(a, "abc", crypto.SecretHint("abc"))
	})
}
