package handlers_test

import (
	"context"
	"encoding/json"
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"testing"

	"github.com/allure-framework/allure-go/testify/require"
	"github.com/google/uuid"

	"github.com/mdg-labs/escalite/services/api/internal/audit"
	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/api/internal/handlers"
)

func seedService(t *testing.T, pool db.DBTX, orgID, teamID uuid.UUID, name string) db.Service {
	t.Helper()

	queries := db.New(pool)
	service, err := queries.CreateService(context.Background(), db.CreateServiceParams{
		ID:             uuid.Must(uuid.NewV7()),
		OrganizationID: orgID,
		TeamID:         teamID,
		Name:           name,
	})
	require.NoError(t, err)
	return service
}

func TestGraphQLEscalationPolicyCRUD(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, pool, cleanup := newTestHandler(t)
		defer cleanup()

		adminCookie := bootstrapAdmin(t, handler)

		queries := db.New(pool)
		admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
		require.NoError(a, err)

		team := seedTeam(t, pool, admin.OrganizationID, "Platform")
		service := seedService(t, pool, admin.OrganizationID, team.ID, "checkout-api")
		adminUserID := admin.ID.String()

		createRec := postGraphQL(t, handler, `mutation {
		createEscalationPolicy(input: {
			serviceId: "`+service.ID.String()+`"
			name: "Default"
			steps: [
				{
					stepOrder: 1
					delayMinutes: 0
					targets: [{ targetType: "user", userId: "`+adminUserID+`" }]
				}
				{
					stepOrder: 2
					delayMinutes: 15
					targets: [{ targetType: "user", userId: "`+adminUserID+`" }]
				}
			]
		}) {
			id
			name
			steps { stepOrder delayMinutes }
		}
	}`, adminCookie)
		require.Equal(a, 200, createRec.Code, createRec.Body.String())

		var createResp struct {
			Data struct {
				CreateEscalationPolicy struct {
					ID    string `json:"id"`
					Name  string `json:"name"`
					Steps []struct {
						StepOrder    int `json:"stepOrder"`
						DelayMinutes int `json:"delayMinutes"`
					} `json:"steps"`
				} `json:"createEscalationPolicy"`
			} `json:"data"`
		}
		require.NoError(a, json.Unmarshal(createRec.Body.Bytes(), &createResp))
		require.Equal(a, "Default", createResp.Data.CreateEscalationPolicy.Name)
		require.Len(a, createResp.Data.CreateEscalationPolicy.Steps, 2)
		require.Equal(a, 1, createResp.Data.CreateEscalationPolicy.Steps[0].StepOrder)
		require.Equal(a, 0, createResp.Data.CreateEscalationPolicy.Steps[0].DelayMinutes)
		require.Equal(a, 2, createResp.Data.CreateEscalationPolicy.Steps[1].StepOrder)
		require.Equal(a, 15, createResp.Data.CreateEscalationPolicy.Steps[1].DelayMinutes)

		policyID := createResp.Data.CreateEscalationPolicy.ID

		events, err := queries.ListAuditEventsByOrganization(context.Background(), admin.OrganizationID)
		require.NoError(a, err)
		require.NotEmpty(a, events)
		foundCreate := false
		for _, event := range events {
			if event.Action == audit.ActionEscalationPolicyCreated {
				foundCreate = true
				break
			}
		}
		require.True(a, foundCreate, "expected escalation_policy.created audit event")

		gapRec := postGraphQL(t, handler, `mutation {
		createEscalationPolicy(input: {
			serviceId: "`+service.ID.String()+`"
			name: "Broken"
			steps: [
				{
					stepOrder: 1
					delayMinutes: 0
					targets: [{ targetType: "user", userId: "`+adminUserID+`" }]
				}
				{ stepOrder: 3, delayMinutes: 5, targets: [] }
			]
		}) { id }
	}`, adminCookie)
		require.Equal(a, 200, gapRec.Code)

		var gapResp struct {
			Errors []struct {
				Extensions map[string]interface{} `json:"extensions"`
			} `json:"errors"`
		}
		require.NoError(a, json.Unmarshal(gapRec.Body.Bytes(), &gapResp))
		require.NotEmpty(a, gapResp.Errors)
		require.Equal(a, handlers.CodeValidation, gapResp.Errors[0].Extensions["code"])

		listRec := postGraphQL(t, handler, `{
		escalationPolicies(serviceId: "`+service.ID.String()+`") {
			id
			name
			steps { stepOrder delayMinutes }
		}
	}`, adminCookie)
		require.Equal(a, 200, listRec.Code, listRec.Body.String())

		var listResp struct {
			Data struct {
				EscalationPolicies []struct {
					ID    string `json:"id"`
					Name  string `json:"name"`
					Steps []struct {
						StepOrder    int `json:"stepOrder"`
						DelayMinutes int `json:"delayMinutes"`
					} `json:"steps"`
				} `json:"escalationPolicies"`
			} `json:"data"`
		}
		require.NoError(a, json.Unmarshal(listRec.Body.Bytes(), &listResp))
		require.Len(a, listResp.Data.EscalationPolicies, 1)
		require.Equal(a, policyID, listResp.Data.EscalationPolicies[0].ID)

		updateRec := postGraphQL(t, handler, `mutation {
		updateEscalationPolicy(input: {
			id: "`+policyID+`"
			name: "Updated"
			steps: [
				{
					stepOrder: 1
					delayMinutes: 0
					targets: [{ targetType: "user", userId: "`+adminUserID+`" }]
				}
				{
					stepOrder: 2
					delayMinutes: 30
					targets: [{ targetType: "user", userId: "`+adminUserID+`" }]
				}
			]
		}) {
			id
			name
			steps { stepOrder delayMinutes }
		}
	}`, adminCookie)
		require.Equal(a, 200, updateRec.Code, updateRec.Body.String())

		var updateResp struct {
			Data struct {
				UpdateEscalationPolicy struct {
					Name  string `json:"name"`
					Steps []struct {
						DelayMinutes int `json:"delayMinutes"`
					} `json:"steps"`
				} `json:"updateEscalationPolicy"`
			} `json:"data"`
		}
		require.NoError(a, json.Unmarshal(updateRec.Body.Bytes(), &updateResp))
		require.Equal(a, "Updated", updateResp.Data.UpdateEscalationPolicy.Name)
		require.Equal(a, 30, updateResp.Data.UpdateEscalationPolicy.Steps[1].DelayMinutes)

		events, err = queries.ListAuditEventsByOrganization(context.Background(), admin.OrganizationID)
		require.NoError(a, err)
		foundUpdate := false
		for _, event := range events {
			if event.Action == audit.ActionEscalationPolicyUpdated {
				foundUpdate = true
				break
			}
		}
		require.True(a, foundUpdate, "expected escalation_policy.updated audit event")

		deleteRec := postGraphQL(t, handler, `mutation {
		deleteEscalationPolicy(id: "`+policyID+`")
	}`, adminCookie)
		require.Equal(a, 200, deleteRec.Code, deleteRec.Body.String())

		var deleteResp struct {
			Data struct {
				DeleteEscalationPolicy bool `json:"deleteEscalationPolicy"`
			} `json:"data"`
		}
		require.NoError(a, json.Unmarshal(deleteRec.Body.Bytes(), &deleteResp))
		require.True(a, deleteResp.Data.DeleteEscalationPolicy)

		events, err = queries.ListAuditEventsByOrganization(context.Background(), admin.OrganizationID)
		require.NoError(a, err)
		foundDelete := false
		for _, event := range events {
			if event.Action == audit.ActionEscalationPolicyDeleted {
				foundDelete = true
				break
			}
		}
		require.True(a, foundDelete, "expected escalation_policy.deleted audit event")
	})
}

func TestGraphQLEscalationPolicyRejectsStepsWithoutTargets(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, pool, cleanup := newTestHandler(t)
		defer cleanup()

		adminCookie := bootstrapAdmin(t, handler)

		queries := db.New(pool)
		admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
		require.NoError(a, err)

		team := seedTeam(t, pool, admin.OrganizationID, "Platform")
		service := seedService(t, pool, admin.OrganizationID, team.ID, "checkout-api")

		rec := postGraphQL(t, handler, `mutation {
		createEscalationPolicy(input: {
			serviceId: "`+service.ID.String()+`"
			name: "No targets"
			steps: [{ stepOrder: 1, delayMinutes: 0, targets: [] }]
		}) { id }
	}`, adminCookie)
		require.Equal(a, 200, rec.Code)

		var resp struct {
			Errors []struct {
				Extensions map[string]interface{} `json:"extensions"`
			} `json:"errors"`
		}
		require.NoError(a, json.Unmarshal(rec.Body.Bytes(), &resp))
		require.NotEmpty(a, resp.Errors)
		require.Equal(a, handlers.CodeValidation, resp.Errors[0].Extensions["code"])
	})
}

func TestGraphQLEscalationPolicyRequiresAdmin(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, pool, cleanup := newTestHandler(t)
		defer cleanup()

		_ = bootstrapAdmin(t, handler)

		queries := db.New(pool)
		admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
		require.NoError(a, err)

		team := seedTeam(t, pool, admin.OrganizationID, "Platform")
		service := seedService(t, pool, admin.OrganizationID, team.ID, "checkout-api")
		member := seedMemberUser(t, pool, admin.OrganizationID, "member@example.com", "member-password-123")
		memberCookie := loginUser(t, handler, member.Email, "member-password-123")

		rec := postGraphQL(t, handler, `mutation {
		createEscalationPolicy(input: {
			serviceId: "`+service.ID.String()+`"
			name: "Default"
			steps: [{
				stepOrder: 1
				delayMinutes: 0
				targets: [{ targetType: "user", userId: "`+member.ID.String()+`" }]
			}]
		}) { id }
	}`, memberCookie)
		require.Equal(a, 200, rec.Code)

		var resp struct {
			Errors []struct {
				Extensions map[string]interface{} `json:"extensions"`
			} `json:"errors"`
		}
		require.NoError(a, json.Unmarshal(rec.Body.Bytes(), &resp))
		require.NotEmpty(a, resp.Errors)
		require.Equal(a, handlers.CodeForbidden, resp.Errors[0].Extensions["code"])
	})
}

func TestGraphQLEscalationPolicyReturnsStepTargets(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, pool, cleanup := newTestHandler(t)
		defer cleanup()

		adminCookie := bootstrapAdmin(t, handler)

		queries := db.New(pool)
		admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
		require.NoError(a, err)

		team := seedTeam(t, pool, admin.OrganizationID, "Platform")
		service := seedService(t, pool, admin.OrganizationID, team.ID, "checkout-api")
		schedule, err := queries.CreateSchedule(context.Background(), db.CreateScheduleParams{
			ID:             uuid.Must(uuid.NewV7()),
			OrganizationID: admin.OrganizationID,
			TeamID:         team.ID,
			Name:           "Primary",
			Timezone:       "UTC",
		})
		require.NoError(a, err)

		adminUserID := admin.ID.String()
		scheduleID := schedule.ID.String()
		webhookURL := "https://example.com/hooks/escalite"

		createRec := postGraphQL(t, handler, `mutation {
		createEscalationPolicy(input: {
			serviceId: "`+service.ID.String()+`"
			name: "With targets"
			steps: [{
				stepOrder: 1
				delayMinutes: 0
				targets: [
					{ targetType: "user", userId: "`+adminUserID+`" }
					{ targetType: "rotation", scheduleId: "`+scheduleID+`" }
					{ targetType: "webhook", webhookUrl: "`+webhookURL+`" }
				]
			}]
		}) {
			id
			steps {
				id
				targets {
					targetType
					userId
					scheduleId
					webhookUrl
				}
			}
		}
	}`, adminCookie)
		require.Equal(a, 200, createRec.Code, createRec.Body.String())

		var createResp struct {
			Data struct {
				CreateEscalationPolicy struct {
					ID    string `json:"id"`
					Steps []struct {
						ID      string `json:"id"`
						Targets []struct {
							TargetType string  `json:"targetType"`
							UserID     *string `json:"userId"`
							ScheduleID *string `json:"scheduleId"`
							WebhookURL *string `json:"webhookUrl"`
						} `json:"targets"`
					} `json:"steps"`
				} `json:"createEscalationPolicy"`
			} `json:"data"`
		}
		require.NoError(a, json.Unmarshal(createRec.Body.Bytes(), &createResp))
		require.Len(a, createResp.Data.CreateEscalationPolicy.Steps, 1)
		require.Len(a, createResp.Data.CreateEscalationPolicy.Steps[0].Targets, 3)

		policyID := createResp.Data.CreateEscalationPolicy.ID

		getRec := postGraphQL(t, handler, `{
		escalationPolicy(id: "`+policyID+`") {
			id
			name
			steps {
				id
				stepOrder
				delayMinutes
				targets {
					id
					targetType
					userId
					scheduleId
					webhookUrl
				}
			}
		}
	}`, adminCookie)
		require.Equal(a, 200, getRec.Code, getRec.Body.String())

		var getResp struct {
			Data struct {
				EscalationPolicy struct {
					ID    string `json:"id"`
					Name  string `json:"name"`
					Steps []struct {
						StepOrder    int `json:"stepOrder"`
						DelayMinutes int `json:"delayMinutes"`
						Targets      []struct {
							ID         string  `json:"id"`
							TargetType string  `json:"targetType"`
							UserID     *string `json:"userId"`
							ScheduleID *string `json:"scheduleId"`
							WebhookURL *string `json:"webhookUrl"`
						} `json:"targets"`
					} `json:"steps"`
				} `json:"escalationPolicy"`
			} `json:"data"`
		}
		require.NoError(a, json.Unmarshal(getRec.Body.Bytes(), &getResp))
		require.Equal(a, policyID, getResp.Data.EscalationPolicy.ID)
		require.Equal(a, "With targets", getResp.Data.EscalationPolicy.Name)
		require.Len(a, getResp.Data.EscalationPolicy.Steps, 1)
		require.Equal(a, 1, getResp.Data.EscalationPolicy.Steps[0].StepOrder)
		require.Len(a, getResp.Data.EscalationPolicy.Steps[0].Targets, 3)

		targetsByType := map[string]struct {
			UserID     *string
			ScheduleID *string
			WebhookURL *string
		}{}
		for _, target := range getResp.Data.EscalationPolicy.Steps[0].Targets {
			require.NotEmpty(a, target.ID)
			targetsByType[target.TargetType] = struct {
				UserID     *string
				ScheduleID *string
				WebhookURL *string
			}{
				UserID:     target.UserID,
				ScheduleID: target.ScheduleID,
				WebhookURL: target.WebhookURL,
			}
		}

		require.NotNil(a, targetsByType["user"].UserID)
		require.Equal(a, adminUserID, *targetsByType["user"].UserID)
		require.NotNil(a, targetsByType["rotation"].ScheduleID)
		require.Equal(a, scheduleID, *targetsByType["rotation"].ScheduleID)
		require.NotNil(a, targetsByType["webhook"].WebhookURL)
		require.Equal(a, webhookURL, *targetsByType["webhook"].WebhookURL)
	})
}
