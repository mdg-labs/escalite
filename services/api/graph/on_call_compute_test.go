package graph

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCurrentOnCallUserDailyRotation(t *testing.T) {
	loc := time.UTC
	anchor := time.Date(2026, 1, 1, 9, 0, 0, 0, loc)
	participants := []string{"user-a", "user-b", "user-c"}

	userID, err := currentOnCallUser("FREQ=DAILY;INTERVAL=1", anchor, loc, time.Date(2026, 1, 3, 12, 0, 0, 0, loc), participants)
	require.NoError(t, err)
	require.Equal(t, "user-c", userID)
}

func TestCurrentOnCallUserDSTSpringForward(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)

	anchor := time.Date(2026, 1, 1, 9, 0, 0, 0, loc)
	participants := []string{"user-a", "user-b", "user-c"}

	// 2026-03-08 is US spring-forward Sunday; 10:00 local is safely after the gap.
	// Jan 1 + 66 days = Mar 8 → occurrence 67 → (67-1) % 3 = 0.
	at := time.Date(2026, 3, 8, 10, 0, 0, 0, loc)
	userID, err := currentOnCallUser("FREQ=DAILY;INTERVAL=1", anchor, loc, at, participants)
	require.NoError(t, err)
	require.Equal(t, "user-a", userID)
}

func TestCurrentOnCallUserDSTFallBack(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)

	anchor := time.Date(2026, 1, 1, 9, 0, 0, 0, loc)
	participants := []string{"user-a", "user-b", "user-c"}

	// 2026-11-01 is US fall-back Sunday; evaluate mid-morning after clocks reset.
	at := time.Date(2026, 11, 1, 10, 0, 0, 0, loc)
	userID, err := currentOnCallUser("FREQ=DAILY;INTERVAL=1", anchor, loc, at, participants)
	require.NoError(t, err)
	require.Equal(t, "user-b", userID)
}

func TestCurrentOnCallUserBeforeAnchor(t *testing.T) {
	loc := time.UTC
	anchor := time.Date(2026, 6, 1, 9, 0, 0, 0, loc)

	_, err := currentOnCallUser("FREQ=DAILY;INTERVAL=1", anchor, loc, time.Date(2026, 5, 1, 12, 0, 0, 0, loc), []string{"user-a"})
	require.ErrorIs(t, err, errNoActiveShift)
}
