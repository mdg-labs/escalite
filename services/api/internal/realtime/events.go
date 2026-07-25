package realtime

import "github.com/google/uuid"

const (
	ChannelAlerts    = "escalite_alerts"
	ChannelSchedules = "escalite_schedules"
)

// AlertEvent is emitted when an alert row is inserted or its status changes.
type AlertEvent struct {
	AlertID        uuid.UUID
	OrganizationID uuid.UUID
	Status         string
	Op             string
}

// ScheduleEvent is emitted when schedule-related rows change.
type ScheduleEvent struct {
	ScheduleID     uuid.UUID
	OrganizationID uuid.UUID
	Table          string
	Op             string
}
