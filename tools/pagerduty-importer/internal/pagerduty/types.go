package pagerduty

import "time"

// User from PagerDuty users API.
type User struct {
	ID       string
	Name     string
	Email    string
	Role     string
	JobTitle string
}

// ScheduleLayer from a PagerDuty schedule.
type ScheduleLayer struct {
	ID                         string
	Name                       string
	RotationTurnLengthSeconds  int
	Users                      []string // user IDs
}

// Schedule from PagerDuty schedules API.
type Schedule struct {
	ID          string
	Name        string
	Description string
	TimeZone    string
	Layers      []ScheduleLayer
}

// Service from PagerDuty services API.
type Service struct {
	ID                 string
	Name               string
	Description        string
	EscalationPolicyID string
}

// EscalationTarget is a rule target on an escalation policy.
type EscalationTarget struct {
	ID   string
	Type string
}

// EscalationRule from a PagerDuty escalation policy.
type EscalationRule struct {
	ID                       string
	EscalationDelayInMinutes int
	Targets                  []EscalationTarget
}

// EscalationPolicy from PagerDuty escalation policies API.
type EscalationPolicy struct {
	ID              string
	Name            string
	Description     string
	NumLoops        int
	EscalationRules []EscalationRule
}

// UnsupportedObject documents a PagerDuty object type not imported.
type UnsupportedObject struct {
	ObjectType string `json:"object_type"`
	SourceID   string `json:"source_id"`
	Name       string `json:"name"`
	Reason     string `json:"reason"`
}

// Counts summarizes source objects for dry-run output.
type Counts struct {
	Users              int
	Schedules          int
	ScheduleLayers     int
	Services           int
	EscalationPolicies int
	EscalationRules    int
	Unsupported        int
}

// Snapshot is a bundled export file (offline import).
type Snapshot struct {
	Users              []User              `json:"users"`
	Schedules          []Schedule          `json:"schedules"`
	Services           []Service           `json:"services"`
	EscalationPolicies []EscalationPolicy  `json:"escalation_policies"`
	Unsupported        []UnsupportedObject `json:"unsupported,omitempty"`
	ExportedAt         time.Time           `json:"exported_at,omitempty"`
}

// Source loads PagerDuty configuration for import.
type Source interface {
	ListUsers() ([]User, error)
	ListSchedules() ([]Schedule, error)
	ListServices() ([]Service, error)
	ListEscalationPolicies() ([]EscalationPolicy, error)
	UnsupportedObjects() []UnsupportedObject
	Counts() Counts
}
