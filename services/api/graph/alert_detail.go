package graph

import (
	"time"

	"github.com/mdg-labs/escalite/services/api/graph/model"
	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/api/internal/escalation"
)

// Custom resolvers for Alert.service, Alert.escalationState, and
// Alert.notificationAttempts live in types.resolvers.go (gqlgen recycles
// implementations into the canonical follow-schema location alongside
// Alert.incident). The helpers below are shared with those resolvers.

func alertEscalationStateFromEngine(state escalation.State) *model.AlertEscalationState {
	var nextEscalationAt *time.Time
	if state.NextEscalationAt != nil {
		t := state.NextEscalationAt.UTC()
		nextEscalationAt = &t
	}

	return &model.AlertEscalationState{
		CurrentStep:        state.CurrentStep,
		NextEscalationAt:   nextEscalationAt,
		EscalatedExhausted: state.EscalatedExhausted,
	}
}

func notificationAttemptFromDB(attempt db.NotificationAttempt) *model.NotificationAttempt {
	var sentAt *time.Time
	if attempt.SentAt.Valid {
		t := attempt.SentAt.Time.UTC()
		sentAt = &t
	}

	return &model.NotificationAttempt{
		ID:        attempt.ID.String(),
		Channel:   attempt.Channel,
		Status:    attempt.Status,
		SentAt:    sentAt,
		CreatedAt: attempt.CreatedAt.Time.UTC(),
	}
}
