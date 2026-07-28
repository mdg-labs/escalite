package auth_test

import (
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"testing"

	"github.com/allure-framework/allure-go/testify/require"

	"github.com/mdg-labs/escalite/services/api/internal/auth"
)

func TestParseBearerToken(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		token, ok := auth.ParseBearerToken("Bearer secret-token")
		require.True(a, ok)
		require.Equal(a, "secret-token", token)

		token, ok = auth.ParseBearerToken("bearer another-token")
		require.True(a, ok)
		require.Equal(a, "another-token", token)

		_, ok = auth.ParseBearerToken("")
		require.False(a, ok)

		_, ok = auth.ParseBearerToken("Basic abc")
		require.False(a, ok)

		_, ok = auth.ParseBearerToken("Bearer")
		require.False(a, ok)
	})
}
