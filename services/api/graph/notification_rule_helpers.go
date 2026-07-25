package graph

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mdg-labs/escalite/services/api/graph/model"
	"github.com/mdg-labs/escalite/services/api/internal/db"
)

type notificationRuleStep struct {
	Channel      string `json:"channel"`
	DelayMinutes int    `json:"delay_minutes"`
}

func alertPriorityToDB(priority model.AlertPriority) (string, error) {
	switch priority {
	case model.AlertPriorityLow:
		return "low", nil
	case model.AlertPriorityHigh:
		return "high", nil
	default:
		return "", fmt.Errorf("invalid alert priority %q", priority)
	}
}

func validateNotificationRuleSteps(steps []*model.NotificationRuleStepInput) ([]notificationRuleStep, error) {
	if len(steps) == 0 {
		return nil, fmt.Errorf("at least one notification rule step is required")
	}

	seen := make(map[string]struct{}, len(steps))
	normalized := make([]notificationRuleStep, 0, len(steps))
	for _, step := range steps {
		if step == nil {
			return nil, fmt.Errorf("notification rule step cannot be null")
		}
		channel := strings.TrimSpace(step.Channel)
		if channel == "" {
			return nil, fmt.Errorf("channel is required")
		}
		if _, exists := seen[channel]; exists {
			return nil, fmt.Errorf("duplicate channel %q in notification rule", channel)
		}
		if err := validateNotificationChannelConfig(channel, map[string]any{}); err != nil {
			return nil, err
		}
		if step.DelayMinutes < 0 {
			return nil, fmt.Errorf("delayMinutes must be >= 0")
		}
		seen[channel] = struct{}{}
		normalized = append(normalized, notificationRuleStep{
			Channel:      channel,
			DelayMinutes: step.DelayMinutes,
		})
	}

	return normalized, nil
}

func notificationRuleStepsToRaw(steps []notificationRuleStep) ([]byte, error) {
	return json.Marshal(steps)
}

func notificationRuleStepsFromRaw(raw []byte) ([]*model.NotificationRuleStep, error) {
	if len(raw) == 0 {
		return []*model.NotificationRuleStep{}, nil
	}
	var steps []notificationRuleStep
	if err := json.Unmarshal(raw, &steps); err != nil {
		return nil, err
	}
	out := make([]*model.NotificationRuleStep, 0, len(steps))
	for _, step := range steps {
		out = append(out, &model.NotificationRuleStep{
			Channel:      step.Channel,
			DelayMinutes: step.DelayMinutes,
		})
	}
	return out, nil
}

func userNotificationRuleFromDB(rule db.UserNotificationRule) (*model.UserNotificationRule, error) {
	steps, err := notificationRuleStepsFromRaw(rule.Steps)
	if err != nil {
		return nil, err
	}
	return &model.UserNotificationRule{
		ID:        rule.ID.String(),
		UserID:    rule.UserID.String(),
		Priority:  alertPriorityFromDB(rule.Priority),
		Steps:     steps,
		CreatedAt: timeFromDB(rule.CreatedAt),
		UpdatedAt: timeFromDB(rule.UpdatedAt),
	}, nil
}