package handlers_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/saltfishpr/ical"
	"github.com/stretchr/testify/require"

	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/api/internal/handlers"
)

func TestScheduleICalExportRequiresAdmin(t *testing.T) {
	handler, pool, cleanup := newTestHandler(t)
	defer cleanup()

	adminCookie := bootstrapAdmin(t, handler)

	queries := db.New(pool)
	admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
	require.NoError(t, err)

	team := seedTeam(t, pool, admin.OrganizationID, "Platform")
	schedule, err := queries.CreateSchedule(context.Background(), db.CreateScheduleParams{
		ID:             uuid.Must(uuid.NewV7()),
		OrganizationID: admin.OrganizationID,
		TeamID:         team.ID,
		Name:           "On-Call",
		Timezone:       "UTC",
	})
	require.NoError(t, err)

	member := seedMemberUser(t, pool, admin.OrganizationID, "member@example.com", "member-password-123")
	memberCookie := loginUser(t, handler, member.Email, "member-password-123")

	memberRec := getScheduleICal(t, handler, schedule.ID.String(), memberCookie)
	require.Equal(t, http.StatusForbidden, memberRec.Code)

	adminRec := getScheduleICal(t, handler, schedule.ID.String(), adminCookie)
	require.Equal(t, http.StatusOK, adminRec.Code)
	require.Equal(t, "text/calendar; charset=utf-8", adminRec.Header().Get("Content-Type"))
}

func TestScheduleICalExportDTStartUsesScheduleTimezone(t *testing.T) {
	handler, pool, cleanup := newTestHandler(t)
	defer cleanup()

	adminCookie := bootstrapAdmin(t, handler)

	queries := db.New(pool)
	admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
	require.NoError(t, err)

	team := seedTeam(t, pool, admin.OrganizationID, "Platform")

	createRec := postGraphQL(t, handler, `mutation {
		createSchedule(input: {
			teamId: "`+team.ID.String()+`"
			name: "DST On-Call"
			timezone: "America/New_York"
		}) { id }
	}`, adminCookie)
	require.Equal(t, 200, createRec.Code, createRec.Body.String())

	var createResp struct {
		Data struct {
			CreateSchedule struct {
				ID string `json:"id"`
			} `json:"createSchedule"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(createRec.Body.Bytes(), &createResp))
	scheduleID := createResp.Data.CreateSchedule.ID

	loc, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)
	anchor := time.Date(2026, 1, 1, 9, 0, 0, 0, loc)

	participants, err := json.Marshal([]string{admin.ID.String()})
	require.NoError(t, err)

	rotationID := uuid.Must(uuid.NewV7())
	_, err = queries.CreateRotation(context.Background(), db.CreateRotationParams{
		ID:             rotationID,
		ScheduleID:     uuid.MustParse(scheduleID),
		OrganizationID: admin.OrganizationID,
		Name:           "Primary",
		Layer:          1,
		Rrule:          "FREQ=DAILY;INTERVAL=1",
		Participants:   participants,
	})
	require.NoError(t, err)

	_, err = pool.Exec(context.Background(),
		`UPDATE rotations SET created_at = $1 WHERE schedule_id = $2`,
		anchor.UTC(), uuid.MustParse(scheduleID),
	)
	require.NoError(t, err)

	rec := getScheduleICal(t, handler, scheduleID, adminCookie)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	body, err := io.ReadAll(rec.Body)
	require.NoError(t, err)

	cal, err := ical.ParseCalendar(body)
	require.NoError(t, err)
	require.Empty(t, cal.Validate())

	events := cal.ComponentsByName(ical.CompVEvent)
	require.NotEmpty(t, events)

	for _, event := range events {
		require.Empty(t, event.Validate())
		dtstart := event.PropertiesByName(ical.PropDTStart)[0]
		require.Equal(t, "America/New_York", dtstart.Params.Get(ical.ParamTZID))
	}

	ics := string(body)
	require.Contains(t, ics, "DTSTART;TZID=America/New_York:")
	require.Contains(t, ics, admin.Email)
	require.Contains(t, ics, "CALNAME:DST On-Call")
}

func getScheduleICal(t *testing.T, handler http.Handler, scheduleID string, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/schedules/"+scheduleID+"/calendar.ics", nil)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func TestScheduleICalExportNotFound(t *testing.T) {
	handler, _, cleanup := newTestHandler(t)
	defer cleanup()

	adminCookie := bootstrapAdmin(t, handler)
	missingID := uuid.Must(uuid.NewV7()).String()

	rec := getScheduleICal(t, handler, missingID, adminCookie)
	require.Equal(t, http.StatusNotFound, rec.Code)

	var errResp struct {
		Code string `json:"code"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &errResp))
	require.Equal(t, handlers.CodeNotFound, errResp.Code)
}

func TestScheduleICalExportUnauthenticated(t *testing.T) {
	handler, _, cleanup := newTestHandler(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/schedules/"+uuid.Must(uuid.NewV7()).String()+"/calendar.ics", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusUnauthorized, rec.Code)
	require.True(t, strings.Contains(rec.Body.String(), handlers.CodeUnauthenticated))
}
