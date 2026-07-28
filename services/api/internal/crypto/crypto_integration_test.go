package crypto_test

import (
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"testing"

	"github.com/allure-framework/allure-go/testify/assert"
	"github.com/allure-framework/allure-go/testify/require"

	"github.com/mdg-labs/escalite/services/api/internal/config"
	"github.com/mdg-labs/escalite/services/api/internal/crypto"
)

func TestEncryptDecryptRoundTripIntegration(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		t.Setenv("ESCALITE_ENCRYPTION_KEY", testKeyHex)

		cfg, err := config.Load(config.Options{
			DefaultListenAddr: ":8080",
		})
		require.NoError(a, err)

		box, err := crypto.NewBox(cfg.EncryptionKey)
		require.NoError(a, err)
		assert.Equal(a, crypto.DefaultKeyID, box.KeyID())

		plaintext := []byte("xoxb-slack-bot-token-1234567890")
		encrypted, err := box.Encrypt(plaintext)
		require.NoError(a, err)
		assert.Equal(a, crypto.DefaultKeyID, encrypted.KeyID)
		assert.NotEmpty(a, encrypted.Ciphertext)
		assert.NotEqual(a, plaintext, encrypted.Ciphertext)

		decrypted, err := box.Decrypt(encrypted)
		require.NoError(a, err)
		assert.Equal(a, plaintext, decrypted)
	})
}

func TestDecryptFailsWithWrongKeyIDIntegration(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		t.Setenv("ESCALITE_ENCRYPTION_KEY", testKeyHex)

		cfg, err := config.Load(config.Options{
			DefaultListenAddr: ":8080",
		})
		require.NoError(a, err)

		box, err := crypto.NewBox(cfg.EncryptionKey)
		require.NoError(a, err)

		encrypted, err := box.Encrypt([]byte("twilio-auth-token"))
		require.NoError(a, err)
		encrypted.KeyID = "v2"

		_, err = box.Decrypt(encrypted)
		require.Error(a, err)
		assert.True(a, crypto.IsKeyIDMismatch(err))
		assert.Contains(a, err.Error(), `encrypted with key id "v2"`)
		assert.Contains(a, err.Error(), `server has "v1" configured`)
		assert.Contains(a, err.Error(), "docs/specs/07-security-and-auth.md")
	})
}
