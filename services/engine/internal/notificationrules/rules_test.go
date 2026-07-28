package notificationrules

import (
	"testing"

	allure "github.com/allure-framework/allure-go/commons/gotest"

	"github.com/allure-framework/allure-go/testify/require"
)

func TestApplyRuleStepsUsesHighPriorityOrdering(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		scheduled := ApplyRuleSteps([]Step{
			{Channel: "push", DelayMinutes: 0},
			{Channel: "email", DelayMinutes: 2},
		}, []string{"email", "push"})

		require.Equal(a, []ScheduledChannel{
			{Channel: "push", DelayMinutes: 0},
			{Channel: "email", DelayMinutes: 2},
		}, scheduled)
	})
}

func TestApplyRuleStepsFiltersToAllowedChannels(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		scheduled := ApplyRuleSteps([]Step{
			{Channel: "push", DelayMinutes: 0},
			{Channel: "slack-dm", DelayMinutes: 1},
			{Channel: "email", DelayMinutes: 2},
		}, []string{"email", "push"})

		require.Equal(a, []ScheduledChannel{
			{Channel: "push", DelayMinutes: 0},
			{Channel: "email", DelayMinutes: 2},
		}, scheduled)
	})
}

func TestDefaultSchedulePreservesStepChannelOrder(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		require.Equal(a, []ScheduledChannel{
			{Channel: "email", DelayMinutes: 0},
			{Channel: "push", DelayMinutes: 0},
		}, DefaultSchedule([]string{"email", "push"}))
	})
}
