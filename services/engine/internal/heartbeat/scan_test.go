package heartbeat

import (
	"testing"

	allure "github.com/allure-framework/allure-go/commons/gotest"

	"github.com/allure-framework/allure-go/testify/require"
)

func TestAlertSourceFromState(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		source, err := AlertSourceFromState([]byte(`{"source":"heartbeat","current_step":1}`))
		require.NoError(a, err)
		require.Equal(a, "heartbeat", source)

		empty, err := AlertSourceFromState(nil)
		require.NoError(a, err)
		require.Empty(a, empty)
	})
}
