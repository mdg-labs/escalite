package schedexport

import (
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/allure-framework/allure-go/testify/require"
	"github.com/google/uuid"
	"github.com/saltfishpr/ical"
)

func TestExportValidatesAgainstRFC5545Fixture(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		rotationID := uuid.MustParse("01950000-0000-7000-8000-000000000001")
		scheduleID := uuid.MustParse("01950000-0000-7000-8000-000000000002")
		timezone := "America/New_York"

		loc, err := time.LoadLocation(timezone)
		require.NoError(a, err)

		anchor := time.Date(2026, 1, 1, 9, 0, 0, 0, loc)
		exportedAt := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
		windowStart := time.Date(2026, 1, 1, 0, 0, 0, 0, loc)
		windowEnd := time.Date(2026, 1, 8, 0, 0, 0, 0, loc)

		ics, err := Export(Input{
			ScheduleID:   scheduleID,
			ScheduleName: "Primary On-Call",
			Timezone:     timezone,
			Rotations: []Rotation{
				{
					ID:             rotationID,
					Name:           "Weekly",
					Layer:          1,
					RRule:          "FREQ=DAILY;INTERVAL=1",
					Anchor:         anchor,
					ParticipantIDs: []string{"user-a", "user-b", "user-c"},
				},
			},
			UserEmails: map[string]string{
				"user-a": "oncall-a@example.com",
				"user-b": "oncall-b@example.com",
				"user-c": "oncall-c@example.com",
			},
			WindowStart: windowStart,
			WindowEnd:   windowEnd,
			ExportedAt:  exportedAt,
		})
		require.NoError(a, err)

		cal, err := ical.ParseCalendar([]byte(ics))
		require.NoError(a, err)

		errs := cal.Validate()
		require.Empty(a, errs, "calendar should validate against RFC 5545")

		events := cal.ComponentsByName(ical.CompVEvent)
		require.Len(a, events, 7)

		for _, event := range events {
			errs := event.Validate()
			require.Empty(a, errs)

			dtstart := event.PropertiesByName(ical.PropDTStart)[0]
			require.Equal(a, timezone, dtstart.Params.Get(ical.ParamTZID))
			require.NotEmpty(a, dtstart.Value)

			dtend := event.PropertiesByName(ical.PropDTEnd)[0]
			require.Equal(a, timezone, dtend.Params.Get(ical.ParamTZID))
		}

		require.Contains(a, ics, "BEGIN:VCALENDAR")
		require.Contains(a, ics, "PRODID:"+prodID)
		require.Contains(a, ics, "CALNAME:Primary On-Call")
		require.Contains(a, ics, "DTSTART;TZID=America/New_York:")
		require.Contains(a, ics, "DTEND;TZID=America/New_York:")
		require.True(a, strings.Contains(ics, "oncall-a@example.com") || strings.Contains(ics, "oncall-b@example.com"))
	})
}

func TestFixtureFileValidatesAgainstRFC5545(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		fixturePath := filepath.Join("testdata", "rfc5545_schedule_fixture.ics")
		data, err := os.ReadFile(fixturePath)
		require.NoError(a, err)

		cal, err := ical.ParseCalendar(data)
		require.NoError(a, err)
		require.Empty(a, cal.Validate())

		event := cal.ComponentsByName(ical.CompVEvent)[0]
		dtstart := event.PropertiesByName(ical.PropDTStart)[0]
		require.Equal(a, "America/New_York", dtstart.Params.Get(ical.ParamTZID))
	})
}

func TestExportRejectsInvalidTimezone(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		_, err := Export(Input{
			ScheduleID:   uuid.Must(uuid.NewV7()),
			ScheduleName: "Broken",
			Timezone:     "Not/A_Timezone",
		})
		require.Error(a, err)
	})
}
