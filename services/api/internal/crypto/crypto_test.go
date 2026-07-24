package crypto_test

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
	_, err := crypto.NewBox([]byte("too-short"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "32 bytes")
}

func TestSecretHint(t *testing.T) {
	assert.Equal(t, "mnop", crypto.SecretHint("abcdefghijklmnop"))
	assert.Equal(t, "abc", crypto.SecretHint("abc"))
}
