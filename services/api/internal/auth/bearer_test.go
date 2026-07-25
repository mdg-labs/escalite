package auth_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mdg-labs/escalite/services/api/internal/auth"
)

func TestParseBearerToken(t *testing.T) {
	token, ok := auth.ParseBearerToken("Bearer secret-token")
	require.True(t, ok)
	require.Equal(t, "secret-token", token)

	token, ok = auth.ParseBearerToken("bearer another-token")
	require.True(t, ok)
	require.Equal(t, "another-token", token)

	_, ok = auth.ParseBearerToken("")
	require.False(t, ok)

	_, ok = auth.ParseBearerToken("Basic abc")
	require.False(t, ok)

	_, ok = auth.ParseBearerToken("Bearer")
	require.False(t, ok)
}
