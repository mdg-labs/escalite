package graph

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mdg-labs/escalite/services/api/internal/gqlerr"
	"github.com/mdg-labs/escalite/services/api/internal/handlers"
)

func TestValidateTimezone(t *testing.T) {
	t.Run("accepts valid IANA timezone", func(t *testing.T) {
		require.NoError(t, validateTimezone("America/New_York"))
	})

	t.Run("rejects invalid timezone", func(t *testing.T) {
		err := validateTimezone("Not/A_Timezone")
		require.Error(t, err)

		var coded *gqlerr.CodedError
		require.ErrorAs(t, err, &coded)
		require.Equal(t, handlers.CodeValidation, coded.Code)
		require.Contains(t, coded.Message, "IANA")
	})

	t.Run("rejects empty timezone", func(t *testing.T) {
		err := validateTimezone("   ")
		require.Error(t, err)

		var coded *gqlerr.CodedError
		require.ErrorAs(t, err, &coded)
		require.Equal(t, handlers.CodeValidation, coded.Code)
	})
}

func TestValidateRRule(t *testing.T) {
	t.Run("accepts valid RRULE", func(t *testing.T) {
		require.NoError(t, validateRRule("FREQ=DAILY;INTERVAL=1"))
	})

	t.Run("rejects invalid RRULE", func(t *testing.T) {
		err := validateRRule("NOT_A_RRULE")
		require.Error(t, err)

		var coded *gqlerr.CodedError
		require.ErrorAs(t, err, &coded)
		require.Equal(t, handlers.CodeValidation, coded.Code)
		require.Contains(t, coded.Message, "rrule is invalid")
	})

	t.Run("rejects empty RRULE", func(t *testing.T) {
		err := validateRRule("")
		require.Error(t, err)

		var coded *gqlerr.CodedError
		require.ErrorAs(t, err, &coded)
		require.Equal(t, handlers.CodeValidation, coded.Code)
	})
}

func TestParseParticipantIDs(t *testing.T) {
	t.Run("rejects empty list", func(t *testing.T) {
		_, err := parseParticipantIDs(nil)
		require.Error(t, err)

		var coded *gqlerr.CodedError
		require.ErrorAs(t, err, &coded)
		require.Equal(t, handlers.CodeValidation, coded.Code)
	})

	t.Run("rejects duplicate ids", func(t *testing.T) {
		id := "018f5a28-9b0e-7000-8000-000000000001"
		_, err := parseParticipantIDs([]string{id, id})
		require.Error(t, err)

		var coded *gqlerr.CodedError
		require.ErrorAs(t, err, &coded)
		require.Equal(t, handlers.CodeValidation, coded.Code)
	})
}
