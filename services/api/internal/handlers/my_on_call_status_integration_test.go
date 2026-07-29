package handlers_test

import (
	"context"
	"encoding/json"
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"net/http"
	"testing"
	"time"

	"github.com/allure-framework/allure-go/testify/require"
	"github.com/google/uuid"

	"github.com/mdg-labs/escalite/services/api/internal/authz"
	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/api/internal/handlers"
)

func TestGraphQLMyOnCallStatusMemberSeesOwnAssignments(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, pool, cleanup := newTestHandler(t)
		defer cleanup()

		adminCookie := bootstrapAdmin(t, handler)

		queries := db.New(pool)
		admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
		require.NoError(a, err)

		orgID := admin.OrganizationID
		team := seedTeam(t, pool, orgID, "Platform")
		onCallMember := seedMemberUser(t, pool, orgID, "oncall-member@example.com", "member-password-123")
		otherMember := seedMemberUser(t, pool, orgID, "other-member@example.com", "member-password-123")
		seedTeamMembership(t, pool, orgID, team.ID, onCallMember.ID)
		seedTeamMembership(t, pool, orgID, team.ID, otherMember.ID)

		createRec := postGraphQL(t, handler, `mutation {
		createSchedule(input: {
			teamId: "`+team.ID.String()+`"
			name: "Primary On-Call"
			timezone: "UTC"
		}) { id }
	}`, adminCookie)
		require.Equal(a, http.StatusOK, createRec.Code, createRec.Body.String())

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
		participants, err := json.Marshal([]string{onCallMember.ID.String()})
		require.NoError(a, err)

		anchor := time.Now().UTC().Add(-48 * time.Hour).Truncate(time.Second)
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

		memberCookie := loginUser(t, handler, onCallMember.Email, "member-password-123")

		rec := postGraphQL(t, handler, `{
		myOnCallStatus {
			scheduleId
			scheduleName
			teamName
			layer
			until
		}
	}`, memberCookie)
		require.Equal(a, http.StatusOK, rec.Code, rec.Body.String())

		var resp struct {
			Data struct {
				MyOnCallStatus []struct {
					ScheduleID   string `json:"scheduleId"`
					ScheduleName string `json:"scheduleName"`
					TeamName     string `json:"teamName"`
					Layer        int    `json:"layer"`
					Until        string `json:"until"`
				} `json:"myOnCallStatus"`
			} `json:"data"`
		}
		require.NoError(a, json.Unmarshal(rec.Body.Bytes(), &resp))
		require.Len(a, resp.Data.MyOnCallStatus, 1)
		require.Equal(a, scheduleID, resp.Data.MyOnCallStatus[0].ScheduleID)
		require.Equal(a, "Primary On-Call", resp.Data.MyOnCallStatus[0].ScheduleName)
		require.Equal(a, "Platform", resp.Data.MyOnCallStatus[0].TeamName)
		require.Equal(a, 1, resp.Data.MyOnCallStatus[0].Layer)
		require.NotEmpty(a, resp.Data.MyOnCallStatus[0].Until)

		otherCookie := loginUser(t, handler, otherMember.Email, "member-password-123")
		otherRec := postGraphQL(t, handler, `{ myOnCallStatus { scheduleId } }`, otherCookie)
		require.Equal(a, http.StatusOK, otherRec.Code, otherRec.Body.String())

		var otherResp struct {
			Data struct {
				MyOnCallStatus []struct {
					ScheduleID string `json:"scheduleId"`
				} `json:"myOnCallStatus"`
			} `json:"data"`
		}
		require.NoError(a, json.Unmarshal(otherRec.Body.Bytes(), &otherResp))
		require.Empty(a, otherResp.Data.MyOnCallStatus)
	})
}

func TestGraphQLMyOnCallStatusRequiresAuth(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, _, cleanup := newTestHandler(t)
		defer cleanup()

		rec := postGraphQL(t, handler, `{ myOnCallStatus { scheduleId } }`, nil)
		require.Equal(a, http.StatusOK, rec.Code)

		var resp struct {
			Errors []struct {
				Extensions map[string]interface{} `json:"extensions"`
			} `json:"errors"`
		}
		require.NoError(a, json.Unmarshal(rec.Body.Bytes(), &resp))
		require.NotEmpty(a, resp.Errors)
		require.Equal(a, handlers.CodeUnauthenticated, resp.Errors[0].Extensions["code"])
	})
}

func TestGraphQLMyOnCallStatusCrossOrgDenied(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, pool, cleanup := newTestHandler(t)
		defer cleanup()

		adminCookie := bootstrapAdmin(t, handler)

		queries := db.New(pool)
		admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
		require.NoError(a, err)

		team := seedTeam(t, pool, admin.OrganizationID, "Platform")
		member := seedMemberUser(t, pool, admin.OrganizationID, "member@example.com", "member-password-123")
		seedTeamMembership(t, pool, admin.OrganizationID, team.ID, member.ID)

		homeScheduleID := uuid.Must(uuid.NewV7())
		_, err = queries.CreateSchedule(context.Background(), db.CreateScheduleParams{
			ID:             homeScheduleID,
			OrganizationID: admin.OrganizationID,
			TeamID:         team.ID,
			Name:           "Home Org Schedule",
			Timezone:       "UTC",
		})
		require.NoError(a, err)

		rotationID := uuid.Must(uuid.NewV7())
		participants, err := json.Marshal([]string{member.ID.String()})
		require.NoError(a, err)
		anchor := time.Now().UTC().Add(-24 * time.Hour).Truncate(time.Second)
		_, err = queries.CreateRotation(context.Background(), db.CreateRotationParams{
			ID:             rotationID,
			ScheduleID:     homeScheduleID,
			OrganizationID: admin.OrganizationID,
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

		otherOrg, err := queries.CreateOrganization(context.Background(), db.CreateOrganizationParams{
			ID:   uuid.Must(uuid.NewV7()),
			Name: "Other Org",
		})
		require.NoError(a, err)

		otherTeam := seedTeam(t, pool, otherOrg.ID, "Other Team")
		_, err = queries.CreateUser(context.Background(), db.CreateUserParams{
			ID:             uuid.Must(uuid.NewV7()),
			AccountID:      admin.AccountID,
			OrganizationID: otherOrg.ID,
			Email:          admin.Email,
			Role:           authz.RoleMember,
		})
		require.NoError(a, err)

		_, err = queries.CreateSchedule(context.Background(), db.CreateScheduleParams{
			ID:             uuid.Must(uuid.NewV7()),
			OrganizationID: otherOrg.ID,
			TeamID:         otherTeam.ID,
			Name:           "Other Org Schedule",
			Timezone:       "UTC",
		})
		require.NoError(a, err)

		switchRec := postGraphQL(t, handler, `mutation {
		switchOrganization(organizationId: "`+otherOrg.ID.String()+`") {
			user { organizationId }
		}
	}`, adminCookie)
		require.Equal(a, http.StatusOK, switchRec.Code, switchRec.Body.String())

		var switchResp struct {
			Data struct {
				SwitchOrganization struct {
					User struct {
						OrganizationID string `json:"organizationId"`
					} `json:"user"`
				} `json:"switchOrganization"`
			} `json:"data"`
		}
		require.NoError(a, json.Unmarshal(switchRec.Body.Bytes(), &switchResp))
		require.Equal(a, otherOrg.ID.String(), switchResp.Data.SwitchOrganization.User.OrganizationID)

		rec := postGraphQL(t, handler, `{
		myOnCallStatus {
			scheduleId
			scheduleName
		}
	}`, adminCookie)
		require.Equal(a, http.StatusOK, rec.Code, rec.Body.String())

		var resp struct {
			Data struct {
				MyOnCallStatus []struct {
					ScheduleID   string `json:"scheduleId"`
					ScheduleName string `json:"scheduleName"`
				} `json:"myOnCallStatus"`
			} `json:"data"`
		}
		require.NoError(a, json.Unmarshal(rec.Body.Bytes(), &resp))
		require.Empty(a, resp.Data.MyOnCallStatus)

		memberCookie := loginUser(t, handler, member.Email, "member-password-123")
		memberRec := postGraphQL(t, handler, `{
		myOnCallStatus {
			scheduleId
			scheduleName
		}
	}`, memberCookie)
		require.Equal(a, http.StatusOK, memberRec.Code, memberRec.Body.String())

		var memberResp struct {
			Data struct {
				MyOnCallStatus []struct {
					ScheduleID   string `json:"scheduleId"`
					ScheduleName string `json:"scheduleName"`
				} `json:"myOnCallStatus"`
			} `json:"data"`
		}
		require.NoError(a, json.Unmarshal(memberRec.Body.Bytes(), &memberResp))
		require.Len(a, memberResp.Data.MyOnCallStatus, 1)
		require.Equal(a, homeScheduleID.String(), memberResp.Data.MyOnCallStatus[0].ScheduleID)
		require.Equal(a, "Home Org Schedule", memberResp.Data.MyOnCallStatus[0].ScheduleName)
	})
}

func TestGraphQLMyOnCallStatusOverrideUntil(t *testing.T) {
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
		seedTeamMembership(t, pool, orgID, team.ID, overrideUser.ID)

		createRec := postGraphQL(t, handler, `mutation {
		createSchedule(input: {
			teamId: "`+team.ID.String()+`"
			name: "Override Schedule"
			timezone: "UTC"
		}) { id }
	}`, adminCookie)
		require.Equal(a, http.StatusOK, createRec.Code, createRec.Body.String())

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

		anchor := time.Now().UTC().Add(-48 * time.Hour).Truncate(time.Second)
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

		evalAt := time.Now().UTC().Truncate(time.Second)
		startsAt := evalAt.Add(-time.Hour).Format(time.RFC3339)
		endsAt := evalAt.Add(2 * time.Hour).Format(time.RFC3339)

		overrideRec := postGraphQL(t, handler, `mutation {
		createOverride(input: {
			scheduleId: "`+scheduleID+`"
			rotationId: "`+rotationID.String()+`"
			userId: "`+overrideUser.ID.String()+`"
			startsAt: "`+startsAt+`"
			endsAt: "`+endsAt+`"
		}) { id }
	}`, adminCookie)
		require.Equal(a, http.StatusOK, overrideRec.Code, overrideRec.Body.String())

		memberCookie := loginUser(t, handler, overrideUser.Email, "member-password-123")
		rec := postGraphQL(t, handler, `{
		myOnCallStatus {
			scheduleId
			layer
			until
		}
	}`, memberCookie)
		require.Equal(a, http.StatusOK, rec.Code, rec.Body.String())

		var resp struct {
			Data struct {
				MyOnCallStatus []struct {
					ScheduleID string `json:"scheduleId"`
					Layer      int    `json:"layer"`
					Until      string `json:"until"`
				} `json:"myOnCallStatus"`
			} `json:"data"`
		}
		require.NoError(a, json.Unmarshal(rec.Body.Bytes(), &resp))
		require.Len(a, resp.Data.MyOnCallStatus, 1)
		require.Equal(a, scheduleID, resp.Data.MyOnCallStatus[0].ScheduleID)
		require.Equal(a, 1, resp.Data.MyOnCallStatus[0].Layer)

		until, err := time.Parse(time.RFC3339, resp.Data.MyOnCallStatus[0].Until)
		require.NoError(a, err)
		expectedUntil, err := time.Parse(time.RFC3339, endsAt)
		require.NoError(a, err)
		require.True(a, until.Equal(expectedUntil))
	})
}
