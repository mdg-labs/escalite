package auth

import (
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"strings"
	"testing"

	"github.com/allure-framework/allure-go/testify/assert"
	"github.com/allure-framework/allure-go/testify/require"
)

func TestHashPasswordRoundTrip(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		password := "correct-horse-battery-staple"

		hash, err := HashPassword(password)
		require.NoError(a, err)
		require.NotEmpty(a, hash)
		assert.True(a, strings.HasPrefix(hash, "$argon2id$"))

		match, err := VerifyPassword(password, hash)
		require.NoError(a, err)
		assert.True(a, match)
	})
}

func TestVerifyPasswordRejectsWrongPassword(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		hash, err := HashPassword("correct-password")
		require.NoError(a, err)

		match, err := VerifyPassword("wrong-password", hash)
		require.NoError(a, err)
		assert.False(a, match)
	})
}
