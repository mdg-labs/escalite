package graph

import (
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"testing"

	"github.com/allure-framework/allure-go/testify/require"

	"github.com/mdg-labs/escalite/services/api/internal/gqlerr"
	"github.com/mdg-labs/escalite/services/api/internal/handlers"
)

func TestValidateNotificationChannelConfig(t *testing.T) {
	allure.Test(t, "rejects unknown channel", func(a *allure.Context) {
		err := validateNotificationChannelConfig("pagerduty", map[string]any{})
		require.Error(a, err)
		require.Contains(a, err.Error(), `unknown notification channel "pagerduty"`)
	})
	allure.Test(t, "accepts registered channel config", func(a *allure.Context) {
		err := validateNotificationChannelConfig("email", map[string]any{})
		require.NoError(a, err)
	})
	allure.Test(t, "rejects invalid push config", func(a *allure.Context) {
		err := validateNotificationChannelConfig("push", map[string]any{})
		require.Error(a, err)

		var coded *gqlerr.CodedError
		_ = coded
		require.Contains(a, err.Error(), "expo_push_token")
	})
	allure.Test(t, "requires channel name", func(a *allure.Context) {
		err := validateNotificationChannelConfig("  ", map[string]any{})
		require.Error(a, err)
		require.Contains(a, err.Error(), "channel is required")
	})
}

func TestValidateNotificationChannelConfigMapsToGraphQLValidation(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		err := validateNotificationChannelConfig("sms", map[string]any{})
		require.Error(a, err)

		wrapped := gqlerr.New(handlers.CodeValidation, err.Error())
		var coded *gqlerr.CodedError
		require.ErrorAs(a, wrapped, &coded)
		require.Equal(a, handlers.CodeValidation, coded.Code)
	})
}
