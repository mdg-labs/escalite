package handlers_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/api/internal/handlers"
)

func TestGraphQLScheduleCRUD(t *testing.T) {
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
			name: "Primary On-Call"
			timezone: "America/New_York"
		}) {
			id
			name
			timezone
			rotations { id }
		}
	}`, adminCookie)
	require.Equal(t, 200, createRec.Code, createRec.Body.String())

	var createResp struct {
		Data struct {
			CreateSchedule struct {
				ID       string `json:"id"`
				Name     string `json:"name"`
				Timezone string `json:"timezone"`
			} `json:"createSchedule"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(createRec.Body.Bytes(), &createResp))
	require.Equal(t, "Primary On-Call", createResp.Data.CreateSchedule.Name)
	require.Equal(t, "America/New_York", createResp.Data.CreateSchedule.Timezone)

	scheduleID := createResp.Data.CreateSchedule.ID

	invalidTZRec := postGraphQL(t, handler, `mutation {
		createSchedule(input: {
			teamId: "`+team.ID.String()+`"
			name: "Bad TZ"
			timezone: "Not/A_Timezone"
		}) { id }
	}`, adminCookie)
	require.Equal(t, 200, invalidTZRec.Code)

	var invalidTZResp struct {
		Errors []struct {
			Message    string                 `json:"message"`
			Extensions map[string]interface{} `json:"extensions"`
		} `json:"errors"`
	}
	require.NoError(t, json.Unmarshal(invalidTZRec.Body.Bytes(), &invalidTZResp))
	require.NotEmpty(t, invalidTZResp.Errors)
	require.Equal(t, handlers.CodeValidation, invalidTZResp.Errors[0].Extensions["code"])
	require.Contains(t, invalidTZResp.Errors[0].Message, "IANA")

	rotationRec := postGraphQL(t, handler, `mutation {
		createRotation(input: {
			scheduleId: "`+scheduleID+`"
			name: "Weekly"
			layer: 1
			rrule: "FREQ=WEEKLY;INTERVAL=1;BYDAY=MO"
			participantIds: ["`+admin.ID.String()+`"]
		}) {
			id
			layer
			rrule
			participantIds
		}
	}`, adminCookie)
	require.Equal(t, 200, rotationRec.Code, rotationRec.Body.String())

	var rotationResp struct {
		Data struct {
			CreateRotation struct {
				ID             string   `json:"id"`
				Layer          int      `json:"layer"`
				Rrule          string   `json:"rrule"`
				ParticipantIds []string `json:"participantIds"`
			} `json:"createRotation"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rotationRec.Body.Bytes(), &rotationResp))
	require.Equal(t, 1, rotationResp.Data.CreateRotation.Layer)
	require.Equal(t, admin.ID.String(), rotationResp.Data.CreateRotation.ParticipantIds[0])

	rotationID := rotationResp.Data.CreateRotation.ID

	invalidRRuleRec := postGraphQL(t, handler, `mutation {
		createRotation(input: {
			scheduleId: "`+scheduleID+`"
			name: "Broken"
			layer: 2
			rrule: "NOT_A_RRULE"
			participantIds: ["`+admin.ID.String()+`"]
		}) { id }
	}`, adminCookie)
	require.Equal(t, 200, invalidRRuleRec.Code)

	var invalidRRuleResp struct {
		Errors []struct {
			Message    string                 `json:"message"`
			Extensions map[string]interface{} `json:"extensions"`
		} `json:"errors"`
	}
	require.NoError(t, json.Unmarshal(invalidRRuleRec.Body.Bytes(), &invalidRRuleResp))
	require.NotEmpty(t, invalidRRuleResp.Errors)
	require.Equal(t, handlers.CodeValidation, invalidRRuleResp.Errors[0].Extensions["code"])
	require.Contains(t, invalidRRuleResp.Errors[0].Message, "rrule is invalid")

	listRec := postGraphQL(t, handler, `{
		schedules(teamId: "`+team.ID.String()+`") {
			id
			name
			rotations { id layer }
		}
	}`, adminCookie)
	require.Equal(t, 200, listRec.Code, listRec.Body.String())

	var listResp struct {
		Data struct {
			Schedules []struct {
				ID        string `json:"id"`
				Name      string `json:"name"`
				Rotations []struct {
					ID    string `json:"id"`
					Layer int    `json:"layer"`
				} `json:"rotations"`
			} `json:"schedules"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(listRec.Body.Bytes(), &listResp))
	require.Len(t, listResp.Data.Schedules, 1)
	require.Equal(t, scheduleID, listResp.Data.Schedules[0].ID)
	require.Len(t, listResp.Data.Schedules[0].Rotations, 1)

	updateScheduleRec := postGraphQL(t, handler, `mutation {
		updateSchedule(input: {
			id: "`+scheduleID+`"
			name: "Updated On-Call"
			timezone: "Europe/Berlin"
		}) {
			id
			name
			timezone
		}
	}`, adminCookie)
	require.Equal(t, 200, updateScheduleRec.Code, updateScheduleRec.Body.String())

	var updateScheduleResp struct {
		Data struct {
			UpdateSchedule struct {
				Name     string `json:"name"`
				Timezone string `json:"timezone"`
			} `json:"updateSchedule"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(updateScheduleRec.Body.Bytes(), &updateScheduleResp))
	require.Equal(t, "Updated On-Call", updateScheduleResp.Data.UpdateSchedule.Name)
	require.Equal(t, "Europe/Berlin", updateScheduleResp.Data.UpdateSchedule.Timezone)

	updateRotationRec := postGraphQL(t, handler, `mutation {
		updateRotation(input: {
			id: "`+rotationID+`"
			name: "Biweekly"
			layer: 1
			rrule: "FREQ=WEEKLY;INTERVAL=2;BYDAY=MO"
			participantIds: ["`+admin.ID.String()+`"]
		}) {
			name
			rrule
		}
	}`, adminCookie)
	require.Equal(t, 200, updateRotationRec.Code, updateRotationRec.Body.String())

	var updateRotationResp struct {
		Data struct {
			UpdateRotation struct {
				Name  string `json:"name"`
				Rrule string `json:"rrule"`
			} `json:"updateRotation"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(updateRotationRec.Body.Bytes(), &updateRotationResp))
	require.Equal(t, "Biweekly", updateRotationResp.Data.UpdateRotation.Name)

	getRec := postGraphQL(t, handler, `{
		schedule(id: "`+scheduleID+`") {
			id
			rotations { id name }
		}
	}`, adminCookie)
	require.Equal(t, 200, getRec.Code, getRec.Body.String())

	deleteRotationRec := postGraphQL(t, handler, `mutation {
		deleteRotation(id: "`+rotationID+`")
	}`, adminCookie)
	require.Equal(t, 200, deleteRotationRec.Code, deleteRotationRec.Body.String())

	deleteScheduleRec := postGraphQL(t, handler, `mutation {
		deleteSchedule(id: "`+scheduleID+`")
	}`, adminCookie)
	require.Equal(t, 200, deleteScheduleRec.Code, deleteScheduleRec.Body.String())

	var deleteScheduleResp struct {
		Data struct {
			DeleteSchedule bool `json:"deleteSchedule"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(deleteScheduleRec.Body.Bytes(), &deleteScheduleResp))
	require.True(t, deleteScheduleResp.Data.DeleteSchedule)

	_, err = queries.GetScheduleByID(context.Background(), db.GetScheduleByIDParams{
		ID:             uuid.MustParse(scheduleID),
		OrganizationID: admin.OrganizationID,
	})
	require.Error(t, err)
}

func TestGraphQLScheduleRequiresAdmin(t *testing.T) {
	handler, pool, cleanup := newTestHandler(t)
	defer cleanup()

	_ = bootstrapAdmin(t, handler)

	queries := db.New(pool)
	admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
	require.NoError(t, err)

	team := seedTeam(t, pool, admin.OrganizationID, "Platform")
	member := seedMemberUser(t, pool, admin.OrganizationID, "member@example.com", "member-password-123")
	memberCookie := loginUser(t, handler, member.Email, "member-password-123")

	rec := postGraphQL(t, handler, `mutation {
		createSchedule(input: {
			teamId: "`+team.ID.String()+`"
			name: "Denied"
			timezone: "UTC"
		}) { id }
	}`, memberCookie)
	require.Equal(t, 200, rec.Code)

	var resp struct {
		Errors []struct {
			Extensions map[string]interface{} `json:"extensions"`
		} `json:"errors"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.NotEmpty(t, resp.Errors)
	require.Equal(t, handlers.CodeForbidden, resp.Errors[0].Extensions["code"])
}
