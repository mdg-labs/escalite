package graph

import (
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"testing"

	"github.com/allure-framework/allure-go/testify/require"

	"github.com/mdg-labs/escalite/services/api/graph/model"
)

func TestValidateNotificationRuleSteps(t *testing.T) {
	allure.Test(t, "requires at least one step", func(a *allure.Context) {
		_, err := validateNotificationRuleSteps(nil)
		require.Error(a, err)
	})
	allure.Test(t, "rejects unknown channel", func(a *allure.Context) {
		_, err := validateNotificationRuleSteps([]*model.NotificationRuleStepInput{
			{Channel: "pagerduty", DelayMinutes: 0},
		})
		require.Error(a, err)
		require.Contains(a, err.Error(), `unknown notification channel "pagerduty"`)
	})
	allure.Test(t, "rejects duplicate channels", func(a *allure.Context) {
		_, err := validateNotificationRuleSteps([]*model.NotificationRuleStepInput{
			{Channel: "email", DelayMinutes: 0},
			{Channel: "email", DelayMinutes: 2},
		})
		require.Error(a, err)
		require.Contains(a, err.Error(), `duplicate channel "email"`)
	})
	allure.Test(t, "accepts ordered registered channels", func(a *allure.Context) {
		steps, err := validateNotificationRuleSteps([]*model.NotificationRuleStepInput{
			{Channel: "push", DelayMinutes: 0},
			{Channel: "email", DelayMinutes: 2},
		})
		require.NoError(a, err)
		require.Equal(a, []notificationRuleStep{
			{Channel: "push", DelayMinutes: 0},
			{Channel: "email", DelayMinutes: 2},
		}, steps)
	})
}

func TestNotificationRuleStepsRoundTrip(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		raw, err := notificationRuleStepsToRaw([]notificationRuleStep{
			{Channel: "push", DelayMinutes: 0},
			{Channel: "email", DelayMinutes: 5},
		})
		require.NoError(a, err)

		steps, err := notificationRuleStepsFromRaw(raw)
		require.NoError(a, err)
		require.Len(a, steps, 2)
		require.Equal(a, "push", steps[0].Channel)
		require.Equal(a, 0, steps[0].DelayMinutes)
		require.Equal(a, "email", steps[1].Channel)
		require.Equal(a, 5, steps[1].DelayMinutes)
	})
}
