package notificationrules

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/mdg-labs/escalite/services/engine/internal/db"
)

// Step is one ordered channel entry in a user notification rule.
type Step struct {
	Channel      string `json:"channel"`
	DelayMinutes int    `json:"delay_minutes"`
}

// ScheduledChannel is a channel to notify after an optional delay.
type ScheduledChannel struct {
	Channel      string
	DelayMinutes int
}

// ParseSteps decodes stored notification rule steps.
func ParseSteps(raw []byte) ([]Step, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var steps []Step
	if err := json.Unmarshal(raw, &steps); err != nil {
		return nil, fmt.Errorf("parse notification rule steps: %w", err)
	}
	return steps, nil
}

// ApplyRuleSteps returns allowed channels in user-rule order with delays.
func ApplyRuleSteps(ruleSteps []Step, allowedChannels []string) []ScheduledChannel {
	allowed := make(map[string]struct{}, len(allowedChannels))
	for _, channel := range allowedChannels {
		allowed[channel] = struct{}{}
	}

	scheduled := make([]ScheduledChannel, 0, len(ruleSteps))
	for _, step := range ruleSteps {
		channel := strings.TrimSpace(step.Channel)
		if channel == "" {
			continue
		}
		if _, ok := allowed[channel]; !ok {
			continue
		}
		delay := step.DelayMinutes
		if delay < 0 {
			delay = 0
		}
		scheduled = append(scheduled, ScheduledChannel{
			Channel:      channel,
			DelayMinutes: delay,
		})
	}
	return scheduled
}

// DefaultSchedule notifies all allowed channels immediately in step order.
func DefaultSchedule(allowedChannels []string) []ScheduledChannel {
	scheduled := make([]ScheduledChannel, 0, len(allowedChannels))
	for _, channel := range allowedChannels {
		scheduled = append(scheduled, ScheduledChannel{
			Channel:      channel,
			DelayMinutes: 0,
		})
	}
	return scheduled
}

// ResolveChannelSchedule loads a user's rule for alert priority and applies it to allowed channels.
func ResolveChannelSchedule(
	ctx context.Context,
	q db.Querier,
	organizationID, userID uuid.UUID,
	alertPriority string,
	allowedChannels []string,
) ([]ScheduledChannel, error) {
	if len(allowedChannels) == 0 {
		return nil, nil
	}

	rule, err := q.GetUserNotificationRuleByPriority(ctx, db.GetUserNotificationRuleByPriorityParams{
		OrganizationID: organizationID,
		UserID:         userID,
		Priority:       strings.ToLower(strings.TrimSpace(alertPriority)),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return DefaultSchedule(allowedChannels), nil
		}
		return nil, fmt.Errorf("load notification rule: %w", err)
	}

	steps, err := ParseSteps(rule.Steps)
	if err != nil {
		return nil, err
	}
	if len(steps) == 0 {
		return DefaultSchedule(allowedChannels), nil
	}

	scheduled := ApplyRuleSteps(steps, allowedChannels)
	if len(scheduled) == 0 {
		return DefaultSchedule(allowedChannels), nil
	}
	return scheduled, nil
}
