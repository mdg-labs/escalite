package jobs

import "github.com/google/uuid"

const (
	EscalationTriggerKind = "escalation_trigger"
	NotifyKind            = "notify"
)

// EscalationTriggerArgs schedules step-1 notifications for a triggered alert.
type EscalationTriggerArgs struct {
	AlertID        uuid.UUID `json:"alert_id"`
	OrganizationID uuid.UUID `json:"organization_id"`
}

func (EscalationTriggerArgs) Kind() string { return EscalationTriggerKind }

// NotifyArgs delivers a single notification attempt.
type NotifyArgs struct {
	NotificationAttemptID uuid.UUID `json:"notification_attempt_id"`
	OrganizationID        uuid.UUID `json:"organization_id"`
}

func (NotifyArgs) Kind() string { return NotifyKind }
