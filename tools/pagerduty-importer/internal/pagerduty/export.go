package pagerduty

import (
	"encoding/json"
	"fmt"
	"os"
)

// ExportReader loads a bundled PagerDuty export JSON file (offline import).
type ExportReader struct {
	snapshot Snapshot
}

// LoadExport reads a PagerDuty export snapshot from disk.
func LoadExport(path string) (*ExportReader, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read export file: %w", err)
	}
	var snapshot Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, fmt.Errorf("parse export file: %w", err)
	}
	return &ExportReader{snapshot: snapshot}, nil
}

func (r *ExportReader) ListUsers() ([]User, error) {
	return append([]User(nil), r.snapshot.Users...), nil
}

func (r *ExportReader) ListSchedules() ([]Schedule, error) {
	return append([]Schedule(nil), r.snapshot.Schedules...), nil
}

func (r *ExportReader) ListServices() ([]Service, error) {
	return append([]Service(nil), r.snapshot.Services...), nil
}

func (r *ExportReader) ListEscalationPolicies() ([]EscalationPolicy, error) {
	return append([]EscalationPolicy(nil), r.snapshot.EscalationPolicies...), nil
}

func (r *ExportReader) UnsupportedObjects() []UnsupportedObject {
	return append([]UnsupportedObject(nil), r.snapshot.Unsupported...)
}

func (r *ExportReader) Counts() Counts {
	layers := 0
	rules := 0
	for _, s := range r.snapshot.Schedules {
		layers += len(s.Layers)
	}
	for _, p := range r.snapshot.EscalationPolicies {
		rules += len(p.EscalationRules)
	}
	return Counts{
		Users:              len(r.snapshot.Users),
		Schedules:          len(r.snapshot.Schedules),
		ScheduleLayers:     layers,
		Services:           len(r.snapshot.Services),
		EscalationPolicies: len(r.snapshot.EscalationPolicies),
		EscalationRules:    rules,
		Unsupported:        len(r.snapshot.Unsupported),
	}
}
