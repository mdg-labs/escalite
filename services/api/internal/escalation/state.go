package escalation

import (
	"encoding/json"
	"fmt"
	"time"
)

// State tracks per-alert escalation progress in alerts.escalation_state.
type State struct {
	CurrentStep            int        `json:"current_step"`
	RepeatCount            int        `json:"repeat_count,omitempty"`
	EscalatedExhausted     bool       `json:"escalated_exhausted,omitempty"`
	NextEscalationAt       *time.Time `json:"next_escalation_at,omitempty"`
	PendingEscalationJobID *int64     `json:"pending_escalation_job_id,omitempty"`
}

func ParseState(raw []byte) (State, error) {
	if len(raw) == 0 {
		return State{}, nil
	}
	var state State
	if err := json.Unmarshal(raw, &state); err != nil {
		return State{}, fmt.Errorf("unmarshal escalation state: %w", err)
	}
	return state, nil
}

func MarshalState(state State) ([]byte, error) {
	raw, err := json.Marshal(state)
	if err != nil {
		return nil, fmt.Errorf("marshal escalation state: %w", err)
	}
	return raw, nil
}

func ClearedTimerState(state State) State {
	state.NextEscalationAt = nil
	state.PendingEscalationJobID = nil
	return state
}
