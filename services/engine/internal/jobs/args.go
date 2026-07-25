package jobs

import "github.com/google/uuid"

const (
	EscalationTriggerKind = "escalation_trigger"
	EscalationStepKind    = "escalation_step"
	NotifyKind            = "notify"
)

// EscalationTriggerArgs schedules step-1 notifications for a triggered alert.
type EscalationTriggerArgs struct {
	AlertID        uuid.UUID `json:"alert_id"`
	OrganizationID uuid.UUID `json:"organization_id"`
}

func (EscalationTriggerArgs) Kind() string { return EscalationTriggerKind }

// EscalationStepArgs fires after a step delay to advance escalation.
type EscalationStepArgs struct {
	AlertID        uuid.UUID `json:"alert_id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	FromStep       int       `json:"from_step"`
}

func (EscalationStepArgs) Kind() string { return EscalationStepKind }

// NotifyArgs delivers a single notification attempt.
type NotifyArgs struct {
	NotificationAttemptID uuid.UUID `json:"notification_attempt_id"`
	OrganizationID        uuid.UUID `json:"organization_id"`
}

func (NotifyArgs) Kind() string { return NotifyKind }

// NotifyMaxAttempts is the River retry budget for outbound notification jobs.
const NotifyMaxAttempts = 3
