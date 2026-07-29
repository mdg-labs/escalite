package oncall

import (
	"errors"
	"fmt"
	"time"

	"github.com/teambition/rrule-go"
)

var errNoActiveShift = errors.New("no active shift")

// CurrentOnCallUser returns the participant on call at instant at for a rotation.
// anchor is the rotation start (typically created_at); shift boundaries use loc.
func CurrentOnCallUser(rruleText string, anchor time.Time, loc *time.Location, at time.Time, participantIDs []string) (string, error) {
	if len(participantIDs) == 0 {
		return "", fmt.Errorf("no participants")
	}

	rule, err := rrule.StrToRRule(rruleText)
	if err != nil {
		return "", err
	}

	anchorInLoc := anchor.In(loc)
	rule.DTStart(anchorInLoc)

	atInLoc := at.In(loc)
	shiftStart := rule.Before(atInLoc, true)
	if shiftStart.IsZero() {
		dtStart := rule.GetDTStart()
		if atInLoc.Before(dtStart) {
			return "", errNoActiveShift
		}
		shiftStart = dtStart
	}

	occurrences := rule.Between(rule.GetDTStart(), shiftStart, true)
	if len(occurrences) == 0 {
		return "", errNoActiveShift
	}

	index := (len(occurrences) - 1) % len(participantIDs)
	return participantIDs[index], nil
}

func IsNoActiveShift(err error) bool {
	return errors.Is(err, errNoActiveShift)
}

// CurrentShiftEndAt returns when the active shift that contains at ends.
// Shift boundaries use loc, matching CurrentOnCallUser.
func CurrentShiftEndAt(rruleText string, anchor time.Time, loc *time.Location, at time.Time) (time.Time, error) {
	rule, err := rrule.StrToRRule(rruleText)
	if err != nil {
		return time.Time{}, err
	}

	anchorInLoc := anchor.In(loc)
	rule.DTStart(anchorInLoc)

	atInLoc := at.In(loc)
	shiftStart := rule.Before(atInLoc, true)
	if shiftStart.IsZero() {
		dtStart := rule.GetDTStart()
		if atInLoc.Before(dtStart) {
			return time.Time{}, errNoActiveShift
		}
		shiftStart = dtStart
	}

	next := rule.After(shiftStart, false)
	if next.IsZero() {
		return time.Time{}, errNoActiveShift
	}

	return next.UTC(), nil
}
