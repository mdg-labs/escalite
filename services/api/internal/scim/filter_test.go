package scim_test

import (
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"testing"

	"github.com/allure-framework/allure-go/testify/require"

	escalitescim "github.com/mdg-labs/escalite/services/api/internal/scim"
)

func TestParseUserNameFilter(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		value, ok := escalitescim.ParseUserNameFilter(`userName eq "alice@example.com"`)
		require.True(a, ok)
		require.Equal(a, "alice@example.com", value)
	})
}

func TestParseDisplayNameFilter(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		value, ok := escalitescim.ParseDisplayNameFilter(`displayName eq "Engineering"`)
		require.True(a, ok)
		require.Equal(a, "Engineering", value)
	})
}

func TestMemberUserIDFromMap(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		value, err := escalitescim.MemberUserID(map[string]any{"value": "550e8400-e29b-41d4-a716-446655440000"})
		require.NoError(a, err)
		require.Equal(a, "550e8400-e29b-41d4-a716-446655440000", value)
	})
}
