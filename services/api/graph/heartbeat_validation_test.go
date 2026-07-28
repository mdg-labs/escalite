package graph

import (
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"testing"

	"github.com/allure-framework/allure-go/testify/require"

	"github.com/mdg-labs/escalite/services/api/internal/gqlerr"
	"github.com/mdg-labs/escalite/services/api/internal/handlers"
)

func TestValidatePositiveDurationSeconds(t *testing.T) {
	allure.Test(t, "accepts positive values", func(a *allure.Context) {
		require.NoError(a, validatePositiveDurationSeconds(60, "intervalSeconds"))
	})
	allure.Test(t, "rejects zero", func(a *allure.Context) {
		err := validatePositiveDurationSeconds(0, "graceSeconds")
		require.Error(a, err)

		var coded *gqlerr.CodedError
		require.ErrorAs(a, err, &coded)
		require.Equal(a, handlers.CodeValidation, coded.Code)
	})
	allure.Test(t, "rejects negative values", func(a *allure.Context) {
		err := validatePositiveDurationSeconds(-1, "intervalSeconds")
		require.Error(a, err)

		var coded *gqlerr.CodedError
		require.ErrorAs(a, err, &coded)
		require.Equal(a, handlers.CodeValidation, coded.Code)
	})
}
