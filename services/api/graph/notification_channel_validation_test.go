package graph

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mdg-labs/escalite/services/api/internal/gqlerr"
	"github.com/mdg-labs/escalite/services/api/internal/handlers"
)

func TestValidateNotificationChannelConfig(t *testing.T) {
	t.Run("rejects unknown channel", func(t *testing.T) {
		err := validateNotificationChannelConfig("pagerduty", map[string]any{})
		require.Error(t, err)
		require.Contains(t, err.Error(), `unknown notification channel "pagerduty"`)
	})

	t.Run("accepts registered channel config", func(t *testing.T) {
		err := validateNotificationChannelConfig("email", map[string]any{})
		require.NoError(t, err)
	})

	t.Run("rejects invalid push config", func(t *testing.T) {
		err := validateNotificationChannelConfig("push", map[string]any{})
		require.Error(t, err)

		var coded *gqlerr.CodedError
		_ = coded
		require.Contains(t, err.Error(), "expo_push_token")
	})

	t.Run("requires channel name", func(t *testing.T) {
		err := validateNotificationChannelConfig("  ", map[string]any{})
		require.Error(t, err)
		require.Contains(t, err.Error(), "channel is required")
	})
}

func TestValidateNotificationChannelConfigMapsToGraphQLValidation(t *testing.T) {
	err := validateNotificationChannelConfig("sms", map[string]any{})
	require.Error(t, err)

	wrapped := gqlerr.New(handlers.CodeValidation, err.Error())
	var coded *gqlerr.CodedError
	require.ErrorAs(t, wrapped, &coded)
	require.Equal(t, handlers.CodeValidation, coded.Code)
}
