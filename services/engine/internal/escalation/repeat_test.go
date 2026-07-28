package escalation

import (
	"testing"

	allure "github.com/allure-framework/allure-go/commons/gotest"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/allure-framework/allure-go/testify/require"
)

func TestCanRepeatLastStep_RepeatBoundary(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		a.T().Parallel()

		maxOne := pgtype.Int4{Int32: 1, Valid: true}
		maxTwo := pgtype.Int4{Int32: 2, Valid: true}

		require.True(a, canRepeatLastStep(true, maxOne, 0))
		require.False(a, canRepeatLastStep(true, maxOne, 1))
		require.True(a, canRepeatLastStep(true, maxTwo, 1))
		require.False(a, canRepeatLastStep(true, maxTwo, 2))

		require.False(a, canRepeatLastStep(false, maxOne, 0))
		require.False(a, canRepeatLastStep(true, pgtype.Int4{}, 0))
	})
}
