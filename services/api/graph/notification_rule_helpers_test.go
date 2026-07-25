package graph

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mdg-labs/escalite/services/api/graph/model"
)

func TestValidateNotificationRuleSteps(t *testing.T) {
	t.Run("requires at least one step", func(t *testing.T) {
		_, err := validateNotificationRuleSteps(nil)
		require.Error(t, err)
	})

	t.Run("rejects unknown channel", func(t *testing.T) {
		_, err := validateNotificationRuleSteps([]*model.NotificationRuleStepInput{
			{Channel: "pagerduty", DelayMinutes: 0},
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), `unknown notification channel "pagerduty"`)
	})

	t.Run("rejects duplicate channels", func(t *testing.T) {
		_, err := validateNotificationRuleSteps([]*model.NotificationRuleStepInput{
			{Channel: "email", DelayMinutes: 0},
			{Channel: "email", DelayMinutes: 2},
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), `duplicate channel "email"`)
	})

	t.Run("accepts ordered registered channels", func(t *testing.T) {
		steps, err := validateNotificationRuleSteps([]*model.NotificationRuleStepInput{
			{Channel: "push", DelayMinutes: 0},
			{Channel: "email", DelayMinutes: 2},
		})
		require.NoError(t, err)
		require.Equal(t, []notificationRuleStep{
			{Channel: "push", DelayMinutes: 0},
			{Channel: "email", DelayMinutes: 2},
		}, steps)
	})
}

func TestNotificationRuleStepsRoundTrip(t *testing.T) {
	raw, err := notificationRuleStepsToRaw([]notificationRuleStep{
		{Channel: "push", DelayMinutes: 0},
		{Channel: "email", DelayMinutes: 5},
	})
	require.NoError(t, err)

	steps, err := notificationRuleStepsFromRaw(raw)
	require.NoError(t, err)
	require.Len(t, steps, 2)
	require.Equal(t, "push", steps[0].Channel)
	require.Equal(t, 0, steps[0].DelayMinutes)
	require.Equal(t, "email", steps[1].Channel)
	require.Equal(t, 5, steps[1].DelayMinutes)
}
