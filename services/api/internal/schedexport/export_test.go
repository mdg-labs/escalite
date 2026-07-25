package schedexport

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/saltfishpr/ical"
	"github.com/stretchr/testify/require"
)

func TestExportValidatesAgainstRFC5545Fixture(t *testing.T) {
	rotationID := uuid.MustParse("01950000-0000-7000-8000-000000000001")
	scheduleID := uuid.MustParse("01950000-0000-7000-8000-000000000002")
	timezone := "America/New_York"

	loc, err := time.LoadLocation(timezone)
	require.NoError(t, err)

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
	require.NoError(t, err)

	cal, err := ical.ParseCalendar([]byte(ics))
	require.NoError(t, err)

	errs := cal.Validate()
	require.Empty(t, errs, "calendar should validate against RFC 5545")

	events := cal.ComponentsByName(ical.CompVEvent)
	require.Len(t, events, 7)

	for _, event := range events {
		errs := event.Validate()
		require.Empty(t, errs)

		dtstart := event.PropertiesByName(ical.PropDTStart)[0]
		require.Equal(t, timezone, dtstart.Params.Get(ical.ParamTZID))
		require.NotEmpty(t, dtstart.Value)

		dtend := event.PropertiesByName(ical.PropDTEnd)[0]
		require.Equal(t, timezone, dtend.Params.Get(ical.ParamTZID))
	}

	require.Contains(t, ics, "BEGIN:VCALENDAR")
	require.Contains(t, ics, "PRODID:"+prodID)
	require.Contains(t, ics, "CALNAME:Primary On-Call")
	require.Contains(t, ics, "DTSTART;TZID=America/New_York:")
	require.Contains(t, ics, "DTEND;TZID=America/New_York:")
	require.True(t, strings.Contains(ics, "oncall-a@example.com") || strings.Contains(ics, "oncall-b@example.com"))
}

func TestFixtureFileValidatesAgainstRFC5545(t *testing.T) {
	fixturePath := filepath.Join("testdata", "rfc5545_schedule_fixture.ics")
	data, err := os.ReadFile(fixturePath)
	require.NoError(t, err)

	cal, err := ical.ParseCalendar(data)
	require.NoError(t, err)
	require.Empty(t, cal.Validate())

	event := cal.ComponentsByName(ical.CompVEvent)[0]
	dtstart := event.PropertiesByName(ical.PropDTStart)[0]
	require.Equal(t, "America/New_York", dtstart.Params.Get(ical.ParamTZID))
}

func TestExportRejectsInvalidTimezone(t *testing.T) {
	_, err := Export(Input{
		ScheduleID:   uuid.Must(uuid.NewV7()),
		ScheduleName: "Broken",
		Timezone:     "Not/A_Timezone",
	})
	require.Error(t, err)
}
