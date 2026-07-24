package crypto_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mdg-labs/escalite/services/api/internal/config"
	"github.com/mdg-labs/escalite/services/api/internal/crypto"
)

func TestEncryptDecryptRoundTripIntegration(t *testing.T) {
	t.Setenv("ESCALITE_ENCRYPTION_KEY", testKeyHex)

	cfg, err := config.Load(config.Options{
		DefaultListenAddr: ":8080",
	})
	require.NoError(t, err)

	box, err := crypto.NewBox(cfg.EncryptionKey)
	require.NoError(t, err)
	assert.Equal(t, crypto.DefaultKeyID, box.KeyID())

	plaintext := []byte("xoxb-slack-bot-token-1234567890")
	encrypted, err := box.Encrypt(plaintext)
	require.NoError(t, err)
	assert.Equal(t, crypto.DefaultKeyID, encrypted.KeyID)
	assert.NotEmpty(t, encrypted.Ciphertext)
	assert.NotEqual(t, plaintext, encrypted.Ciphertext)

	decrypted, err := box.Decrypt(encrypted)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestDecryptFailsWithWrongKeyIDIntegration(t *testing.T) {
	t.Setenv("ESCALITE_ENCRYPTION_KEY", testKeyHex)

	cfg, err := config.Load(config.Options{
		DefaultListenAddr: ":8080",
	})
	require.NoError(t, err)

	box, err := crypto.NewBox(cfg.EncryptionKey)
	require.NoError(t, err)

	encrypted, err := box.Encrypt([]byte("twilio-auth-token"))
	require.NoError(t, err)
	encrypted.KeyID = "v2"

	_, err = box.Decrypt(encrypted)
	require.Error(t, err)
	assert.True(t, crypto.IsKeyIDMismatch(err))
	assert.Contains(t, err.Error(), `encrypted with key id "v2"`)
	assert.Contains(t, err.Error(), `server has "v1" configured`)
	assert.Contains(t, err.Error(), "docs/specs/07-security-and-auth.md")
}
