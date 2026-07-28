package graph

import (
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"testing"

	"github.com/allure-framework/allure-go/testify/require"

	"github.com/mdg-labs/escalite/services/api/internal/gqlerr"
	"github.com/mdg-labs/escalite/services/api/internal/handlers"
)

func TestValidateTimezone(t *testing.T) {
	allure.Test(t, "accepts valid IANA timezone", func(a *allure.Context) {
		require.NoError(a, validateTimezone("America/New_York"))
	})
	allure.Test(t, "rejects invalid timezone", func(a *allure.Context) {
		err := validateTimezone("Not/A_Timezone")
		require.Error(a, err)

		var coded *gqlerr.CodedError
		require.ErrorAs(a, err, &coded)
		require.Equal(a, handlers.CodeValidation, coded.Code)
		require.Contains(a, coded.Message, "IANA")
	})
	allure.Test(t, "rejects empty timezone", func(a *allure.Context) {
		err := validateTimezone("   ")
		require.Error(a, err)

		var coded *gqlerr.CodedError
		require.ErrorAs(a, err, &coded)
		require.Equal(a, handlers.CodeValidation, coded.Code)
	})
}

func TestValidateRRule(t *testing.T) {
	allure.Test(t, "accepts valid RRULE", func(a *allure.Context) {
		require.NoError(a, validateRRule("FREQ=DAILY;INTERVAL=1"))
	})
	allure.Test(t, "rejects invalid RRULE", func(a *allure.Context) {
		err := validateRRule("NOT_A_RRULE")
		require.Error(a, err)

		var coded *gqlerr.CodedError
		require.ErrorAs(a, err, &coded)
		require.Equal(a, handlers.CodeValidation, coded.Code)
		require.Contains(a, coded.Message, "rrule is invalid")
	})
	allure.Test(t, "rejects empty RRULE", func(a *allure.Context) {
		err := validateRRule("")
		require.Error(a, err)

		var coded *gqlerr.CodedError
		require.ErrorAs(a, err, &coded)
		require.Equal(a, handlers.CodeValidation, coded.Code)
	})
}

func TestParseParticipantIDs(t *testing.T) {
	allure.Test(t, "rejects empty list", func(a *allure.Context) {
		_, err := parseParticipantIDs(nil)
		require.Error(a, err)

		var coded *gqlerr.CodedError
		require.ErrorAs(a, err, &coded)
		require.Equal(a, handlers.CodeValidation, coded.Code)
	})
	allure.Test(t, "rejects duplicate ids", func(a *allure.Context) {
		id := "018f5a28-9b0e-7000-8000-000000000001"
		_, err := parseParticipantIDs([]string{id, id})
		require.Error(a, err)

		var coded *gqlerr.CodedError
		require.ErrorAs(a, err, &coded)
		require.Equal(a, handlers.CodeValidation, coded.Code)
	})
}
