package auth

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashPasswordRoundTrip(t *testing.T) {
	password := "correct-horse-battery-staple"

	hash, err := HashPassword(password)
	require.NoError(t, err)
	require.NotEmpty(t, hash)
	assert.True(t, strings.HasPrefix(hash, "$argon2id$"))

	match, err := VerifyPassword(password, hash)
	require.NoError(t, err)
	assert.True(t, match)
}

func TestVerifyPasswordRejectsWrongPassword(t *testing.T) {
	hash, err := HashPassword("correct-password")
	require.NoError(t, err)

	match, err := VerifyPassword("wrong-password", hash)
	require.NoError(t, err)
	assert.False(t, match)
}
