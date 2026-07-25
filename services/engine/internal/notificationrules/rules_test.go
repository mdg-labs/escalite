package notificationrules

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestApplyRuleStepsUsesHighPriorityOrdering(t *testing.T) {
	scheduled := ApplyRuleSteps([]Step{
		{Channel: "push", DelayMinutes: 0},
		{Channel: "email", DelayMinutes: 2},
	}, []string{"email", "push"})

	require.Equal(t, []ScheduledChannel{
		{Channel: "push", DelayMinutes: 0},
		{Channel: "email", DelayMinutes: 2},
	}, scheduled)
}

func TestApplyRuleStepsFiltersToAllowedChannels(t *testing.T) {
	scheduled := ApplyRuleSteps([]Step{
		{Channel: "push", DelayMinutes: 0},
		{Channel: "slack-dm", DelayMinutes: 1},
		{Channel: "email", DelayMinutes: 2},
	}, []string{"email", "push"})

	require.Equal(t, []ScheduledChannel{
		{Channel: "push", DelayMinutes: 0},
		{Channel: "email", DelayMinutes: 2},
	}, scheduled)
}

func TestDefaultSchedulePreservesStepChannelOrder(t *testing.T) {
	require.Equal(t, []ScheduledChannel{
		{Channel: "email", DelayMinutes: 0},
		{Channel: "push", DelayMinutes: 0},
	}, DefaultSchedule([]string{"email", "push"}))
}
