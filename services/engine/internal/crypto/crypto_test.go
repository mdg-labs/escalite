package crypto_test

import (
	"encoding/hex"
	"testing"

	allure "github.com/allure-framework/allure-go/commons/gotest"
	"github.com/allure-framework/allure-go/testify/assert"
	"github.com/allure-framework/allure-go/testify/require"

	"github.com/mdg-labs/escalite/services/engine/internal/crypto"
)

const testKeyHex = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func TestEncryptDecryptRoundTrip(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		key, err := hex.DecodeString(testKeyHex)
		require.NoError(a, err)

		box, err := crypto.NewBox(key)
		require.NoError(a, err)

		plaintext := []byte("twilio-auth-token")
		encrypted, err := box.Encrypt(plaintext)
		require.NoError(a, err)
		assert.Equal(a, crypto.DefaultKeyID, encrypted.KeyID)
		assert.NotEmpty(a, encrypted.Ciphertext)

		decrypted, err := box.Decrypt(encrypted)
		require.NoError(a, err)
		assert.Equal(a, plaintext, decrypted)
	})
}
