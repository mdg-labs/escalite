package crypto_test

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mdg-labs/escalite/services/engine/internal/crypto"
)

const testKeyHex = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key, err := hex.DecodeString(testKeyHex)
	require.NoError(t, err)

	box, err := crypto.NewBox(key)
	require.NoError(t, err)

	plaintext := []byte("twilio-auth-token")
	encrypted, err := box.Encrypt(plaintext)
	require.NoError(t, err)
	assert.Equal(t, crypto.DefaultKeyID, encrypted.KeyID)
	assert.NotEmpty(t, encrypted.Ciphertext)

	decrypted, err := box.Decrypt(encrypted)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}
