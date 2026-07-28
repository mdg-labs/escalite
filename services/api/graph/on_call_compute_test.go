package graph

import (
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"testing"
	"time"

	"github.com/allure-framework/allure-go/testify/require"

	"github.com/mdg-labs/escalite/services/engine/oncall"
)

func TestCurrentOnCallUserDailyRotation(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		loc := time.UTC
		anchor := time.Date(2026, 1, 1, 9, 0, 0, 0, loc)
		participants := []string{"user-a", "user-b", "user-c"}

		userID, err := oncall.CurrentOnCallUser("FREQ=DAILY;INTERVAL=1", anchor, loc, time.Date(2026, 1, 3, 12, 0, 0, 0, loc), participants)
		require.NoError(a, err)
		require.Equal(a, "user-c", userID)
	})
}

func TestCurrentOnCallUserDSTSpringForward(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		loc, err := time.LoadLocation("America/New_York")
		require.NoError(a, err)

		anchor := time.Date(2026, 1, 1, 9, 0, 0, 0, loc)
		participants := []string{"user-a", "user-b", "user-c"}

		// 2026-03-08 is US spring-forward Sunday; 10:00 local is safely after the gap.
		// Jan 1 + 66 days = Mar 8 → occurrence 67 → (67-1) % 3 = 0.
		at := time.Date(2026, 3, 8, 10, 0, 0, 0, loc)
		userID, err := oncall.CurrentOnCallUser("FREQ=DAILY;INTERVAL=1", anchor, loc, at, participants)
		require.NoError(a, err)
		require.Equal(a, "user-a", userID)
	})
}

func TestCurrentOnCallUserDSTFallBack(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		loc, err := time.LoadLocation("America/New_York")
		require.NoError(a, err)

		anchor := time.Date(2026, 1, 1, 9, 0, 0, 0, loc)
		participants := []string{"user-a", "user-b", "user-c"}

		// 2026-11-01 is US fall-back Sunday; evaluate mid-morning after clocks reset.
		at := time.Date(2026, 11, 1, 10, 0, 0, 0, loc)
		userID, err := oncall.CurrentOnCallUser("FREQ=DAILY;INTERVAL=1", anchor, loc, at, participants)
		require.NoError(a, err)
		require.Equal(a, "user-b", userID)
	})
}

func TestCurrentOnCallUserBeforeAnchor(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		loc := time.UTC
		anchor := time.Date(2026, 6, 1, 9, 0, 0, 0, loc)

		_, err := oncall.CurrentOnCallUser("FREQ=DAILY;INTERVAL=1", anchor, loc, time.Date(2026, 5, 1, 12, 0, 0, 0, loc), []string{"user-a"})
		require.True(a, oncall.IsNoActiveShift(err))
	})
}
