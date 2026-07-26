package mapping

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
)

// File is the audit record of PagerDuty source IDs to Escalite UUIDs.
type File struct {
	ImportedAt         time.Time         `json:"imported_at"`
	OrganizationID     uuid.UUID         `json:"organization_id"`
	TeamID             uuid.UUID         `json:"team_id"`
	Users              map[string]string `json:"users"`
	Schedules          map[string]string `json:"schedules"`
	Rotations          map[string]string `json:"rotations"`
	Services           map[string]string `json:"services"`
	EscalationPolicies map[string]string `json:"escalation_policies"`
	EscalationSteps    map[string]string `json:"escalation_steps"`
	Skipped            []SkippedRecord   `json:"skipped,omitempty"`
}

// SkippedRecord documents entities that could not be imported.
type SkippedRecord struct {
	EntityType string `json:"entity_type"`
	SourceID   string `json:"source_id"`
	Reason     string `json:"reason"`
}

// Store tracks ID mappings during import.
type Store struct {
	OrganizationID     uuid.UUID
	TeamID           uuid.UUID
	Users            map[string]uuid.UUID
	Schedules        map[string]uuid.UUID
	Rotations        map[string]uuid.UUID
	Services         map[string]uuid.UUID
	EscalationPolicies map[string]uuid.UUID
	EscalationSteps  map[string]uuid.UUID
	Skipped          []SkippedRecord
}

func NewStore(orgID, teamID uuid.UUID) *Store {
	return &Store{
		OrganizationID:     orgID,
		TeamID:             teamID,
		Users:              make(map[string]uuid.UUID),
		Schedules:          make(map[string]uuid.UUID),
		Rotations:          make(map[string]uuid.UUID),
		Services:           make(map[string]uuid.UUID),
		EscalationPolicies: make(map[string]uuid.UUID),
		EscalationSteps:    make(map[string]uuid.UUID),
	}
}

func (s *Store) AddSkipped(entityType, sourceID, reason string) {
	s.Skipped = append(s.Skipped, SkippedRecord{
		EntityType: entityType,
		SourceID:   sourceID,
		Reason:     reason,
	})
}

func (s *Store) ToFile(importedAt time.Time) File {
	return File{
		ImportedAt:         importedAt,
		OrganizationID:     s.OrganizationID,
		TeamID:             s.TeamID,
		Users:              uuidMapToString(s.Users),
		Schedules:          uuidMapToString(s.Schedules),
		Rotations:          uuidMapToString(s.Rotations),
		Services:           uuidMapToString(s.Services),
		EscalationPolicies: uuidMapToString(s.EscalationPolicies),
		EscalationSteps:    uuidMapToString(s.EscalationSteps),
		Skipped:            s.Skipped,
	}
}

func uuidMapToString(m map[string]uuid.UUID) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v.String()
	}
	return out
}

// WriteJSON writes the mapping file with indented JSON.
func WriteJSON(path string, file File) error {
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal mapping: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write mapping file: %w", err)
	}
	return nil
}
