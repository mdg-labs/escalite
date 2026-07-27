package graph

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOrganizationUsersListLimit(t *testing.T) {
	t.Parallel()

	defaultLimit := defaultOrganizationUsersLimit
	require.Equal(t, int32(500), organizationUsersListLimit(nil))
	require.Equal(t, int32(500), organizationUsersListLimit(&defaultLimit))

	zero := 0
	require.Equal(t, int32(500), organizationUsersListLimit(&zero))

	custom := 100
	require.Equal(t, int32(100), organizationUsersListLimit(&custom))

	overMax := 1000
	require.Equal(t, int32(500), organizationUsersListLimit(&overMax))
}
