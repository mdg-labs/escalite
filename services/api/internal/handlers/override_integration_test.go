package handlers_test

import (
	"context"
	"encoding/json"
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"testing"
	"time"

	"github.com/allure-framework/allure-go/testify/require"
	"github.com/google/uuid"

	"github.com/mdg-labs/escalite/services/api/internal/audit"
	"github.com/mdg-labs/escalite/services/api/internal/db"
)

func TestGraphQLOverrideCRUDAndOnCallPrecedence(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, pool, cleanup := newTestHandler(t)
		defer cleanup()

		adminCookie := bootstrapAdmin(t, handler)

		queries := db.New(pool)
		admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
		require.NoError(a, err)

		orgID := admin.OrganizationID
		team := seedTeam(t, pool, orgID, "Platform")
		overrideUser := seedMemberUser(t, pool, orgID, "override-user@example.com", "member-password-123")

		createRec := postGraphQL(t, handler, `mutation {
		createSchedule(input: {
			teamId: "`+team.ID.String()+`"
			name: "Override Schedule"
			timezone: "UTC"
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

		rotationID := uuid.Must(uuid.NewV7())
		participants, err := json.Marshal([]string{admin.ID.String()})
		require.NoError(a, err)

		anchor := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		_, err = queries.CreateRotation(context.Background(), db.CreateRotationParams{
			ID:             rotationID,
			ScheduleID:     uuid.MustParse(scheduleID),
			OrganizationID: orgID,
			Name:           "Primary",
			Layer:          1,
			Rrule:          "FREQ=DAILY;INTERVAL=1",
			Participants:   participants,
		})
		require.NoError(a, err)
		_, err = pool.Exec(context.Background(),
			`UPDATE rotations SET created_at = $1 WHERE id = $2`,
			anchor, rotationID,
		)
		require.NoError(a, err)

		evalAt := time.Date(2026, 1, 5, 12, 0, 0, 0, time.UTC)
		startsAt := evalAt.Add(-time.Hour).Format(time.RFC3339)
		endsAt := evalAt.Add(time.Hour).Format(time.RFC3339)

		overrideRec := postGraphQL(t, handler, `mutation {
		createOverride(input: {
			scheduleId: "`+scheduleID+`"
			rotationId: "`+rotationID.String()+`"
			userId: "`+overrideUser.ID.String()+`"
			startsAt: "`+startsAt+`"
			endsAt: "`+endsAt+`"
		}) {
			id
			userId
			replacedUserId
		}
	}`, adminCookie)
		require.Equal(a, 200, overrideRec.Code, overrideRec.Body.String())

		var overrideResp struct {
			Data struct {
				CreateOverride struct {
					ID             string  `json:"id"`
					UserID         string  `json:"userId"`
					ReplacedUserID *string `json:"replacedUserId"`
				} `json:"createOverride"`
			} `json:"data"`
		}
		require.NoError(a, json.Unmarshal(overrideRec.Body.Bytes(), &overrideResp))
		require.Equal(a, overrideUser.ID.String(), overrideResp.Data.CreateOverride.UserID)
		require.NotNil(a, overrideResp.Data.CreateOverride.ReplacedUserID)
		require.Equal(a, admin.ID.String(), *overrideResp.Data.CreateOverride.ReplacedUserID)

		overrideID := overrideResp.Data.CreateOverride.ID

		events, err := queries.ListAuditEventsByOrganization(context.Background(), orgID)
		require.NoError(a, err)
		foundCreate := false
		for _, event := range events {
			if event.Action == audit.ActionOverrideCreated {
				foundCreate = true
				break
			}
		}
		require.True(a, foundCreate, "expected override.created audit event")

		listRec := postGraphQL(t, handler, `{
		overrides(scheduleId: "`+scheduleID+`") {
			id
			userId
		}
	}`, adminCookie)
		require.Equal(a, 200, listRec.Code, listRec.Body.String())

		var listResp struct {
			Data struct {
				Overrides []struct {
					ID     string `json:"id"`
					UserID string `json:"userId"`
				} `json:"overrides"`
			} `json:"data"`
		}
		require.NoError(a, json.Unmarshal(listRec.Body.Bytes(), &listResp))
		require.Len(a, listResp.Data.Overrides, 1)
		require.Equal(a, overrideID, listResp.Data.Overrides[0].ID)

		onCallRec := postGraphQL(t, handler, `{
		onCallNow(scheduleId: "`+scheduleID+`", at: "`+evalAt.Format(time.RFC3339)+`") {
			layers { layer userId }
		}
	}`, adminCookie)
		require.Equal(a, 200, onCallRec.Code, onCallRec.Body.String())

		var onCallResp struct {
			Data struct {
				OnCallNow struct {
					Layers []struct {
						Layer  int    `json:"layer"`
						UserID string `json:"userId"`
					} `json:"layers"`
				} `json:"onCallNow"`
			} `json:"data"`
		}
		require.NoError(a, json.Unmarshal(onCallRec.Body.Bytes(), &onCallResp))
		require.Len(a, onCallResp.Data.OnCallNow.Layers, 1)
		require.Equal(a, overrideUser.ID.String(), onCallResp.Data.OnCallNow.Layers[0].UserID)

		deleteRec := postGraphQL(t, handler, `mutation {
		deleteOverride(id: "`+overrideID+`")
	}`, adminCookie)
		require.Equal(a, 200, deleteRec.Code, deleteRec.Body.String())

		var deleteResp struct {
			Data struct {
				DeleteOverride bool `json:"deleteOverride"`
			} `json:"data"`
		}
		require.NoError(a, json.Unmarshal(deleteRec.Body.Bytes(), &deleteResp))
		require.True(a, deleteResp.Data.DeleteOverride)

		events, err = queries.ListAuditEventsByOrganization(context.Background(), orgID)
		require.NoError(a, err)
		foundDelete := false
		for _, event := range events {
			if event.Action == audit.ActionOverrideDeleted {
				foundDelete = true
				break
			}
		}
		require.True(a, foundDelete, "expected override.deleted audit event")

		softDeleted, err := queries.GetOverrideByID(context.Background(), db.GetOverrideByIDParams{
			ID:             uuid.MustParse(overrideID),
			OrganizationID: orgID,
		})
		require.Error(a, err)
		require.Equal(a, uuid.Nil, softDeleted.ID)

		row := pool.QueryRow(context.Background(),
			`SELECT deleted_at IS NOT NULL FROM overrides WHERE id = $1`,
			uuid.MustParse(overrideID),
		)
		var isDeleted bool
		require.NoError(a, row.Scan(&isDeleted))
		require.True(a, isDeleted)

		afterDeleteRec := postGraphQL(t, handler, `{
		onCallNow(scheduleId: "`+scheduleID+`", at: "`+evalAt.Format(time.RFC3339)+`") {
			layers { layer userId }
		}
	}`, adminCookie)
		require.Equal(a, 200, afterDeleteRec.Code, afterDeleteRec.Body.String())

		var afterDeleteResp struct {
			Data struct {
				OnCallNow struct {
					Layers []struct {
						UserID string `json:"userId"`
					} `json:"layers"`
				} `json:"onCallNow"`
			} `json:"data"`
		}
		require.NoError(a, json.Unmarshal(afterDeleteRec.Body.Bytes(), &afterDeleteResp))
		require.Len(a, afterDeleteResp.Data.OnCallNow.Layers, 1)
		require.Equal(a, admin.ID.String(), afterDeleteResp.Data.OnCallNow.Layers[0].UserID)
	})
}

func TestGraphQLOverrideCreateValidation(t *testing.T) {
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
			Name:           "Validation",
			Timezone:       "UTC",
		})
		require.NoError(a, err)

		rotationID := uuid.Must(uuid.NewV7())
		participants, err := json.Marshal([]string{admin.ID.String()})
		require.NoError(a, err)
		_, err = queries.CreateRotation(context.Background(), db.CreateRotationParams{
			ID:             rotationID,
			ScheduleID:     schedule.ID,
			OrganizationID: admin.OrganizationID,
			Name:           "Primary",
			Layer:          1,
			Rrule:          "FREQ=DAILY;INTERVAL=1",
			Participants:   participants,
		})
		require.NoError(a, err)

		otherScheduleID := uuid.Must(uuid.NewV7())
		_, err = queries.CreateSchedule(context.Background(), db.CreateScheduleParams{
			ID:             otherScheduleID,
			OrganizationID: admin.OrganizationID,
			TeamID:         team.ID,
			Name:           "Other",
			Timezone:       "UTC",
		})
		require.NoError(a, err)

		startsAt := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)
		endsAt := time.Date(2026, 2, 1, 1, 0, 0, 0, time.UTC).Format(time.RFC3339)

		mismatchRec := postGraphQL(t, handler, `mutation {
		createOverride(input: {
			scheduleId: "`+otherScheduleID.String()+`"
			rotationId: "`+rotationID.String()+`"
			userId: "`+admin.ID.String()+`"
			startsAt: "`+startsAt+`"
			endsAt: "`+endsAt+`"
		}) { id }
	}`, adminCookie)
		require.Equal(a, 200, mismatchRec.Code)

		var mismatchResp struct {
			Errors []struct {
				Message string `json:"message"`
			} `json:"errors"`
		}
		require.NoError(a, json.Unmarshal(mismatchRec.Body.Bytes(), &mismatchResp))
		require.NotEmpty(a, mismatchResp.Errors)
		require.Contains(a, mismatchResp.Errors[0].Message, "schedule")

		invalidWindowRec := postGraphQL(t, handler, `mutation {
		createOverride(input: {
			scheduleId: "`+schedule.ID.String()+`"
			rotationId: "`+rotationID.String()+`"
			userId: "`+admin.ID.String()+`"
			startsAt: "`+endsAt+`"
			endsAt: "`+startsAt+`"
		}) { id }
	}`, adminCookie)
		require.Equal(a, 200, invalidWindowRec.Code)

		var invalidWindowResp struct {
			Errors []struct {
				Message string `json:"message"`
			} `json:"errors"`
		}
		require.NoError(a, json.Unmarshal(invalidWindowRec.Body.Bytes(), &invalidWindowResp))
		require.NotEmpty(a, invalidWindowResp.Errors)
		require.Contains(a, invalidWindowResp.Errors[0].Message, "endsAt")
	})
}

func TestGraphQLOverrideRequiresAdmin(t *testing.T) {
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
		overrides(scheduleId: "`+schedule.ID.String()+`") { id }
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
		overrides(scheduleId: "`+schedule.ID.String()+`") { id }
	}`, adminCookie)
		require.Equal(a, 200, allowed.Code)

		var allowedResp struct {
			Data struct {
				Overrides []struct {
					ID string `json:"id"`
				} `json:"overrides"`
			} `json:"data"`
		}
		require.NoError(a, json.Unmarshal(allowed.Body.Bytes(), &allowedResp))
		require.Empty(a, allowedResp.Data.Overrides)
	})
}
