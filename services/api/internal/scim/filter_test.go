package scim_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	escalitescim "github.com/mdg-labs/escalite/services/api/internal/scim"
)

func TestParseUserNameFilter(t *testing.T) {
	value, ok := escalitescim.ParseUserNameFilter(`userName eq "alice@example.com"`)
	require.True(t, ok)
	require.Equal(t, "alice@example.com", value)
}

func TestParseDisplayNameFilter(t *testing.T) {
	value, ok := escalitescim.ParseDisplayNameFilter(`displayName eq "Engineering"`)
	require.True(t, ok)
	require.Equal(t, "Engineering", value)
}

func TestMemberUserIDFromMap(t *testing.T) {
	value, err := escalitescim.MemberUserID(map[string]any{"value": "550e8400-e29b-41d4-a716-446655440000"})
	require.NoError(t, err)
	require.Equal(t, "550e8400-e29b-41d4-a716-446655440000", value)
}
