package graph

import (
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"testing"

	"github.com/allure-framework/allure-go/testify/require"
)

func TestOrganizationUsersListLimit(t *testing.T) {
	t.Parallel()
	allure.Wrap(t, func(a *allure.Context) {

		defaultLimit := defaultOrganizationUsersLimit
		require.Equal(a, int32(500), organizationUsersListLimit(nil))
		require.Equal(a, int32(500), organizationUsersListLimit(&defaultLimit))

		zero := 0
		require.Equal(a, int32(500), organizationUsersListLimit(&zero))

		custom := 100
		require.Equal(a, int32(100), organizationUsersListLimit(&custom))

		overMax := 1000
		require.Equal(a, int32(500), organizationUsersListLimit(&overMax))
	})
}
