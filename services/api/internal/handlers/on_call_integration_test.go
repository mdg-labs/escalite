package handlers_test

import (
	"context"
	"encoding/json"
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"testing"
	"time"

	"github.com/allure-framework/allure-go/testify/require"
	"github.com/google/uuid"

	"github.com/mdg-labs/escalite/services/api/internal/db"
)

func TestGraphQLOnCallNowReturnsPrimaryAndSecondaryLayers(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, pool, cleanup := newTestHandler(t)
		defer cleanup()

		adminCookie := bootstrapAdmin(t, handler)

		queries := db.New(pool)
		admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
		require.NoError(a, err)

		orgID := admin.OrganizationID
		team := seedTeam(t, pool, orgID, "Platform")

		userB := seedMemberUser(t, pool, orgID, "oncall-b@example.com", "member-password-123")
		userC := seedMemberUser(t, pool, orgID, "oncall-c@example.com", "member-password-123")
		userD := seedMemberUser(t, pool, orgID, "oncall-d@example.com", "member-password-123")

		createRec := postGraphQL(t, handler, `mutation {
		createSchedule(input: {
			teamId: "`+team.ID.String()+`"
			name: "DST On-Call"
			timezone: "America/New_York"
		}) { id }
	}`, adminCookie)
		require.Equal(a, 200, createRec.Code, createRec.Body.String())

		var createResp struct {
			Data struct {
				CreateSchedule struct {
					ID string `json:"id"`
				} `json:"createSchedule"`
			} `json:"data"`
		}
		require.NoError(a, json.Unmarshal(createRec.Body.Bytes(), &createResp))
		scheduleID := createResp.Data.CreateSchedule.ID

		loc, err := time.LoadLocation("America/New_York")
		require.NoError(a, err)
		anchor := time.Date(2026, 1, 1, 9, 0, 0, 0, loc)

		primaryRotationID := uuid.Must(uuid.NewV7())
		secondaryRotationID := uuid.Must(uuid.NewV7())
		participantsPrimary, err := json.Marshal([]string{admin.ID.String(), userB.ID.String(), userC.ID.String()})
		require.NoError(a, err)
		participantsSecondary, err := json.Marshal([]string{userC.ID.String(), userD.ID.String()})
		require.NoError(a, err)

		_, err = queries.CreateRotation(context.Background(), db.CreateRotationParams{
			ID:             primaryRotationID,
			ScheduleID:     uuid.MustParse(scheduleID),
			OrganizationID: orgID,
			Name:           "Primary",
			Layer:          1,
			Rrule:          "FREQ=DAILY;INTERVAL=1",
			Participants:   participantsPrimary,
		})
		require.NoError(a, err)
		_, err = queries.CreateRotation(context.Background(), db.CreateRotationParams{
			ID:             secondaryRotationID,
			ScheduleID:     uuid.MustParse(scheduleID),
			OrganizationID: orgID,
			Name:           "Secondary",
			Layer:          2,
			Rrule:          "FREQ=DAILY;INTERVAL=1",
			Participants:   participantsSecondary,
		})
		require.NoError(a, err)

		_, err = pool.Exec(context.Background(),
			`UPDATE rotations SET created_at = $1 WHERE schedule_id = $2`,
			anchor.UTC(), uuid.MustParse(scheduleID),
		)
		require.NoError(a, err)

		springForwardAt := time.Date(2026, 3, 8, 10, 0, 0, 0, loc).UTC().Format(time.RFC3339)
		springRec := postGraphQL(t, handler, `{
		onCallNow(scheduleId: "`+scheduleID+`", at: "`+springForwardAt+`") {
			scheduleId
			layers { layer rotationId userId }
		}
	}`, adminCookie)
		require.Equal(a, 200, springRec.Code, springRec.Body.String())

		var springResp struct {
			Data struct {
				OnCallNow struct {
					ScheduleID string `json:"scheduleId"`
					Layers     []struct {
						Layer      int    `json:"layer"`
						RotationID string `json:"rotationId"`
						UserID     string `json:"userId"`
					} `json:"layers"`
				} `json:"onCallNow"`
			} `json:"data"`
		}
		require.NoError(a, json.Unmarshal(springRec.Body.Bytes(), &springResp))
		require.Equal(a, scheduleID, springResp.Data.OnCallNow.ScheduleID)
		require.Len(a, springResp.Data.OnCallNow.Layers, 2)

		layersByNumber := map[int]string{}
		for _, layer := range springResp.Data.OnCallNow.Layers {
			layersByNumber[layer.Layer] = layer.UserID
		}
		require.Equal(a, admin.ID.String(), layersByNumber[1])
		require.Equal(a, userC.ID.String(), layersByNumber[2])

		fallBackAt := time.Date(2026, 11, 1, 10, 0, 0, 0, loc).UTC().Format(time.RFC3339)
		fallRec := postGraphQL(t, handler, `{
		onCallNow(scheduleId: "`+scheduleID+`", at: "`+fallBackAt+`") {
			layers { layer userId }
		}
	}`, adminCookie)
		require.Equal(a, 200, fallRec.Code, fallRec.Body.String())

		var fallResp struct {
			Data struct {
				OnCallNow struct {
					Layers []struct {
						Layer  int    `json:"layer"`
						UserID string `json:"userId"`
					} `json:"layers"`
				} `json:"onCallNow"`
			} `json:"data"`
		}
		require.NoError(a, json.Unmarshal(fallRec.Body.Bytes(), &fallResp))
		require.Len(a, fallResp.Data.OnCallNow.Layers, 2)

		layersByNumber = map[int]string{}
		for _, layer := range fallResp.Data.OnCallNow.Layers {
			layersByNumber[layer.Layer] = layer.UserID
		}
		require.Equal(a, userB.ID.String(), layersByNumber[1])
		require.Equal(a, userC.ID.String(), layersByNumber[2])
	})
}

func TestGraphQLOnCallNowRequiresAdmin(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, pool, cleanup := newTestHandler(t)
		defer cleanup()

		adminCookie := bootstrapAdmin(t, handler)

		queries := db.New(pool)
		admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
		require.NoError(a, err)

		team := seedTeam(t, pool, admin.OrganizationID, "Platform")
		schedule, err := queries.CreateSchedule(context.Background(), db.CreateScheduleParams{
			ID:             uuid.Must(uuid.NewV7()),
			OrganizationID: admin.OrganizationID,
			TeamID:         team.ID,
			Name:           "On-Call",
			Timezone:       "UTC",
		})
		require.NoError(a, err)

		member := seedMemberUser(t, pool, admin.OrganizationID, "member@example.com", "member-password-123")
		memberCookie := loginUser(t, handler, member.Email, "member-password-123")

		rec := postGraphQL(t, handler, `{
		onCallNow(scheduleId: "`+schedule.ID.String()+`") { scheduleId }
	}`, memberCookie)
		require.Equal(a, 200, rec.Code)

		var resp struct {
			Errors []struct {
				Extensions map[string]interface{} `json:"extensions"`
			} `json:"errors"`
		}
		require.NoError(a, json.Unmarshal(rec.Body.Bytes(), &resp))
		require.NotEmpty(a, resp.Errors)

		allowed := postGraphQL(t, handler, `{
		onCallNow(scheduleId: "`+schedule.ID.String()+`") { scheduleId }
	}`, adminCookie)
		require.Equal(a, 200, allowed.Code)

		var allowedResp struct {
			Data struct {
				OnCallNow *struct {
					ScheduleID string `json:"scheduleId"`
				} `json:"onCallNow"`
			} `json:"data"`
		}
		require.NoError(a, json.Unmarshal(allowed.Body.Bytes(), &allowedResp))
		require.NotNil(a, allowedResp.Data.OnCallNow)
		require.Equal(a, schedule.ID.String(), allowedResp.Data.OnCallNow.ScheduleID)
	})
}
