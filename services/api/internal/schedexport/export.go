package schedexport

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/saltfishpr/ical"
	"github.com/teambition/rrule-go"
)

const (
	prodID          = "-//Escalite//Schedule Export//EN"
	defaultExportDays = 90
)

// Rotation describes one schedule layer for ICS export.
type Rotation struct {
	ID             uuid.UUID
	Name           string
	Layer          int
	RRule          string
	Anchor         time.Time
	ParticipantIDs []string
}

// Input configures schedule calendar export.
type Input struct {
	ScheduleID   uuid.UUID
	ScheduleName string
	Timezone     string
	Rotations    []Rotation
	UserEmails   map[string]string
	WindowStart  time.Time
	WindowEnd    time.Time
	ExportedAt   time.Time
}

// Export builds an RFC 5545 VCALENDAR document for on-call rotation shifts.
func Export(in Input) (string, error) {
	if in.ScheduleID == uuid.Nil {
		return "", fmt.Errorf("schedule id is required")
	}
	if in.Timezone == "" {
		return "", fmt.Errorf("timezone is required")
	}

	loc, err := time.LoadLocation(in.Timezone)
	if err != nil {
		return "", fmt.Errorf("load timezone: %w", err)
	}

	exportedAt := in.ExportedAt
	if exportedAt.IsZero() {
		exportedAt = time.Now().UTC()
	}

	windowStart := in.WindowStart
	windowEnd := in.WindowEnd
	if windowStart.IsZero() && windowEnd.IsZero() {
		windowStart = exportedAt
		windowEnd = exportedAt.AddDate(0, 0, defaultExportDays)
	}
	if !windowEnd.After(windowStart) {
		return "", fmt.Errorf("export window end must be after start")
	}

	cal := ical.NewCalendar(prodID)
	if in.ScheduleName != "" {
		cal.AddText("CALNAME", in.ScheduleName)
	}

	for _, rotation := range in.Rotations {
		events, err := rotationEvents(rotation, loc, in.Timezone, in.UserEmails, windowStart, windowEnd, exportedAt)
		if err != nil {
			return "", fmt.Errorf("rotation %s: %w", rotation.ID, err)
		}
		for _, event := range events {
			cal.AddComponent(event)
		}
	}

	return cal.String(), nil
}

func rotationEvents(
	rotation Rotation,
	loc *time.Location,
	timezone string,
	userEmails map[string]string,
	windowStart, windowEnd, exportedAt time.Time,
) ([]*ical.Component, error) {
	if len(rotation.ParticipantIDs) == 0 {
		return nil, nil
	}

	rule, err := rrule.StrToRRule(rotation.RRule)
	if err != nil {
		return nil, err
	}

	anchorInLoc := rotation.Anchor.In(loc)
	rule.DTStart(anchorInLoc)

	dtStart := rule.GetDTStart()
	occurrenceEnd := windowEnd.AddDate(0, 0, 7)
	occurrences := rule.Between(dtStart, occurrenceEnd, true)
	if len(occurrences) == 0 {
		return nil, nil
	}

	events := make([]*ical.Component, 0, len(occurrences))
	for i, start := range occurrences {
		end, ok := shiftEnd(rule, occurrences, i, windowEnd)
		if !ok {
			continue
		}
		if !end.After(windowStart) || !start.Before(windowEnd) {
			continue
		}

		participantID := rotation.ParticipantIDs[i%len(rotation.ParticipantIDs)]
		summaryUser := userEmails[participantID]
		if summaryUser == "" {
			summaryUser = participantID
		}

		startLocal := start.In(loc)
		endLocal := end.In(loc)
		uid := fmt.Sprintf("%s-%d@escalite", rotation.ID, startLocal.Unix())

		event := ical.NewEvent(uid, exportedAt.UTC()).
			AddDateTimeWithTZID(ical.PropDTStart, startLocal, timezone).
			AddDateTimeWithTZID(ical.PropDTEnd, endLocal, timezone).
			AddText(ical.PropSummary, fmt.Sprintf("Layer %d: %s - %s", rotation.Layer, rotation.Name, summaryUser))

		events = append(events, event)
	}

	return events, nil
}

func shiftEnd(rule *rrule.RRule, occurrences []time.Time, index int, windowEnd time.Time) (time.Time, bool) {
	start := occurrences[index]
	var end time.Time
	if index+1 < len(occurrences) {
		end = occurrences[index+1]
	} else {
		next := rule.After(start, false)
		if next.IsZero() {
			end = windowEnd
		} else {
			end = next
		}
	}
	if !end.After(start) {
		return time.Time{}, false
	}
	return end, true
}
