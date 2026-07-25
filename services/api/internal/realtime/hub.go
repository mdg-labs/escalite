package realtime

import (
	"sync"

	"github.com/google/uuid"
)

type alertSubscription struct {
	organizationID uuid.UUID
	ch             chan AlertEvent
}

type scheduleSubscription struct {
	organizationID uuid.UUID
	ch             chan ScheduleEvent
}

// Hub fans out database NOTIFY events to in-process subscribers.
// GraphQL subscription resolvers (#97) subscribe per organization.
type Hub struct {
	mu        sync.RWMutex
	nextID    int
	alertSubs map[int]alertSubscription
	schedSubs map[int]scheduleSubscription
}

// NewHub returns an empty realtime event hub.
func NewHub() *Hub {
	return &Hub{
		alertSubs: make(map[int]alertSubscription),
		schedSubs: make(map[int]scheduleSubscription),
	}
}

// SubscribeAlerts registers for alert events scoped to organizationID.
// The returned cancel function removes the subscription.
func (h *Hub) SubscribeAlerts(organizationID uuid.UUID) (<-chan AlertEvent, func()) {
	h.mu.Lock()
	defer h.mu.Unlock()

	id := h.nextID
	h.nextID++
	ch := make(chan AlertEvent, 16)
	h.alertSubs[id] = alertSubscription{
		organizationID: organizationID,
		ch:             ch,
	}

	cancel := func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		sub, ok := h.alertSubs[id]
		if !ok {
			return
		}
		delete(h.alertSubs, id)
		close(sub.ch)
	}

	return ch, cancel
}

// SubscribeSchedules registers for schedule events scoped to organizationID.
func (h *Hub) SubscribeSchedules(organizationID uuid.UUID) (<-chan ScheduleEvent, func()) {
	h.mu.Lock()
	defer h.mu.Unlock()

	id := h.nextID
	h.nextID++
	ch := make(chan ScheduleEvent, 16)
	h.schedSubs[id] = scheduleSubscription{
		organizationID: organizationID,
		ch:             ch,
	}

	cancel := func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		sub, ok := h.schedSubs[id]
		if !ok {
			return
		}
		delete(h.schedSubs, id)
		close(sub.ch)
	}

	return ch, cancel
}

// PublishAlert delivers an alert event to matching subscribers.
func (h *Hub) PublishAlert(event AlertEvent) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, sub := range h.alertSubs {
		if sub.organizationID != event.OrganizationID {
			continue
		}
		select {
		case sub.ch <- event:
		default:
		}
	}
}

// PublishSchedule delivers a schedule event to matching subscribers.
func (h *Hub) PublishSchedule(event ScheduleEvent) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, sub := range h.schedSubs {
		if sub.organizationID != event.OrganizationID {
			continue
		}
		select {
		case sub.ch <- event:
		default:
		}
	}
}
