package realtime

import (
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"testing"
	"time"

	"github.com/allure-framework/allure-go/testify/require"
	"github.com/google/uuid"
)

func TestHubPublishAlertScopedByOrganization(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		hub := NewHub()
		orgA := uuid.Must(uuid.NewV7())
		orgB := uuid.Must(uuid.NewV7())

		chA, cancelA := hub.SubscribeAlerts(orgA)
		defer cancelA()
		chB, cancelB := hub.SubscribeAlerts(orgB)
		defer cancelB()

		event := AlertEvent{
			AlertID:        uuid.Must(uuid.NewV7()),
			OrganizationID: orgA,
			Status:         "acknowledged",
			Op:             "UPDATE",
		}
		hub.PublishAlert(event)

		select {
		case got := <-chA:
			require.Equal(a, event, got)
		case <-time.After(time.Second):
			t.Fatal("expected alert event on orgA subscriber")
		}

		select {
		case <-chB:
			t.Fatal("orgB subscriber should not receive orgA event")
		case <-time.After(50 * time.Millisecond):
		}
	})
}

func TestHubPublishScheduleScopedByOrganization(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		hub := NewHub()
		orgID := uuid.Must(uuid.NewV7())
		otherOrg := uuid.Must(uuid.NewV7())

		ch, cancel := hub.SubscribeSchedules(orgID)
		defer cancel()
		otherCh, cancelOther := hub.SubscribeSchedules(otherOrg)
		defer cancelOther()

		event := ScheduleEvent{
			ScheduleID:     uuid.Must(uuid.NewV7()),
			OrganizationID: orgID,
			Table:          "rotations",
			Op:             "INSERT",
		}
		hub.PublishSchedule(event)

		select {
		case got := <-ch:
			require.Equal(a, event, got)
		case <-time.After(time.Second):
			t.Fatal("expected schedule event")
		}

		select {
		case <-otherCh:
			t.Fatal("other org subscriber should not receive event")
		case <-time.After(50 * time.Millisecond):
		}
	})
}

func TestNextBackoff(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		require.Equal(a, 500*time.Millisecond, nextBackoff(250*time.Millisecond, 10*time.Second))
		require.Equal(a, 10*time.Second, nextBackoff(8*time.Second, 10*time.Second))
		require.Equal(a, 10*time.Second, nextBackoff(10*time.Second, 10*time.Second))
	})
}

func TestHubPublishTimelineScopedByIncident(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		hub := NewHub()
		incidentA := uuid.Must(uuid.NewV7())
		incidentB := uuid.Must(uuid.NewV7())

		chA, cancelA := hub.SubscribeTimeline(incidentA)
		defer cancelA()
		chB, cancelB := hub.SubscribeTimeline(incidentB)
		defer cancelB()

		event := TimelineEvent{
			TimelineEventID: uuid.Must(uuid.NewV7()),
			IncidentID:      incidentA,
			OrganizationID:  uuid.Must(uuid.NewV7()),
			EventType:       "note",
			Op:              "INSERT",
		}
		hub.PublishTimeline(event)

		select {
		case got := <-chA:
			require.Equal(a, event, got)
		case <-time.After(time.Second):
			t.Fatal("expected timeline event on incidentA subscriber")
		}

		select {
		case <-chB:
			t.Fatal("incidentB subscriber should not receive incidentA event")
		case <-time.After(50 * time.Millisecond):
		}
	})
}
