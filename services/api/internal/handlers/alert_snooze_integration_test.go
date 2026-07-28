package handlers_test

import (
	"context"
	"encoding/json"
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"testing"
	"time"

	"github.com/allure-framework/allure-go/testify/require"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/mdg-labs/escalite/services/api/internal/audit"
	"github.com/mdg-labs/escalite/services/api/internal/db"
	apiescalation "github.com/mdg-labs/escalite/services/api/internal/escalation"
)

func seedEscalationPolicyWithTargets(
	t *testing.T,
	pool db.DBTX,
	orgID, adminID, serviceID uuid.UUID,
	step1DelayMinutes int32,
) {
	t.Helper()

	queries := db.New(pool)
	policyID := uuid.Must(uuid.NewV7())
	step1ID := uuid.Must(uuid.NewV7())
	step2ID := uuid.Must(uuid.NewV7())

	_, err := queries.CreateEscalationPolicy(context.Background(), db.CreateEscalationPolicyParams{
		ID:             policyID,
		OrganizationID: orgID,
		ServiceID:      serviceID,
		Name:           "Default",
	})
	require.NoError(t, err)

	_, err = queries.CreateEscalationStep(context.Background(), db.CreateEscalationStepParams{
		ID:                 step1ID,
		EscalationPolicyID: policyID,
		OrganizationID:     orgID,
		StepOrder:          1,
		DelayMinutes:       step1DelayMinutes,
		RepeatLastStep:     false,
		MaxRepeats:         pgtype.Int4{},
	})
	require.NoError(t, err)

	_, err = queries.CreateEscalationStep(context.Background(), db.CreateEscalationStepParams{
		ID:                 step2ID,
		EscalationPolicyID: policyID,
		OrganizationID:     orgID,
		StepOrder:          2,
		DelayMinutes:       5,
		RepeatLastStep:     false,
		MaxRepeats:         pgtype.Int4{},
	})
	require.NoError(t, err)

	channels, err := json.Marshal([]string{"email"})
	require.NoError(t, err)

	_, err = queries.CreateEscalationStepTarget(context.Background(), db.CreateEscalationStepTargetParams{
		ID:               uuid.Must(uuid.NewV7()),
		EscalationStepID: step1ID,
		OrganizationID:   orgID,
		TargetType:       "user",
		UserID:           pgtype.UUID{Bytes: adminID, Valid: true},
		Channels:         channels,
	})
	require.NoError(t, err)

	_, err = queries.CreateEscalationStepTarget(context.Background(), db.CreateEscalationStepTargetParams{
		ID:               uuid.Must(uuid.NewV7()),
		EscalationStepID: step2ID,
		OrganizationID:   orgID,
		TargetType:       "user",
		UserID:           pgtype.UUID{Bytes: adminID, Valid: true},
		Channels:         channels,
	})
	require.NoError(t, err)
}

func TestGraphQLSnoozeAndReEscalateAlert(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, pool, cleanup := newTestHandler(t)
		defer cleanup()

		adminCookie := bootstrapAdmin(t, handler)

		queries := db.New(pool)
		admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
		require.NoError(a, err)

		team := seedTeam(t, pool, admin.OrganizationID, "Platform")
		service := seedService(t, pool, admin.OrganizationID, team.ID, "checkout-api")
		seedEscalationPolicyWithTargets(t, pool, admin.OrganizationID, admin.ID, service.ID, 30)

		ctx := context.Background()
		nextAt := time.Now().Add(30 * time.Minute).UTC()
		stateRaw, err := apiescalation.MarshalState(apiescalation.State{
			CurrentStep:      1,
			NextEscalationAt: &nextAt,
		})
		require.NoError(a, err)

		alert, err := queries.CreateTriggeredAlert(ctx, db.CreateTriggeredAlertParams{
			ID:               uuid.Must(uuid.NewV7()),
			OrganizationID:   admin.OrganizationID,
			ServiceID:        service.ID,
			IntegrationKeyID: pgtype.UUID{},
			DedupKey:         "graphql-snooze",
			Summary:          "GraphQL snooze target",
			Priority:         "high",
			EscalationState:  stateRaw,
		})
		require.NoError(a, err)

		snoozeRec := postGraphQL(t, handler, `mutation {
		snoozeAlert(id: "`+alert.ID.String()+`", durationMinutes: 10) {
			id
			status
		}
	}`, adminCookie)
		require.Equal(a, 200, snoozeRec.Code)

		var snoozeResp struct {
			Data struct {
				SnoozeAlert struct {
					ID     string `json:"id"`
					Status string `json:"status"`
				} `json:"snoozeAlert"`
			} `json:"data"`
			Errors []struct {
				Message string `json:"message"`
			} `json:"errors"`
		}
		require.NoError(a, json.Unmarshal(snoozeRec.Body.Bytes(), &snoozeResp))
		require.Empty(a, snoozeResp.Errors)
		require.Equal(a, "TRIGGERED", snoozeResp.Data.SnoozeAlert.Status)

		after, err := queries.GetAlertByID(ctx, db.GetAlertByIDParams{
			ID:             alert.ID,
			OrganizationID: admin.OrganizationID,
		})
		require.NoError(a, err)
		afterState, err := apiescalation.ParseState(after.EscalationState)
		require.NoError(a, err)
		require.NotNil(a, afterState.NextEscalationAt)
		expected := nextAt.Add(10 * time.Minute)
		require.WithinDuration(a, expected, *afterState.NextEscalationAt, 2*time.Second)

		events, err := queries.ListAuditEventsByOrganization(ctx, admin.OrganizationID)
		require.NoError(a, err)
		var snoozeEvent *db.AuditEvent
		for i := range events {
			if events[i].Action == audit.ActionAlertEscalationSnoozed {
				snoozeEvent = &events[i]
				break
			}
		}
		require.NotNil(a, snoozeEvent)
		require.Equal(a, alert.ID, uuid.UUID(snoozeEvent.TargetID.Bytes))

		ackRec := postGraphQL(t, handler, `mutation {
		acknowledgeAlert(id: "`+alert.ID.String()+`") { id status }
	}`, adminCookie)
		require.Equal(a, 200, ackRec.Code)

		reEscalateRec := postGraphQL(t, handler, `mutation {
		reEscalateAlert(id: "`+alert.ID.String()+`") {
			id
			status
		}
	}`, adminCookie)
		require.Equal(a, 200, reEscalateRec.Code)

		var reEscalateResp struct {
			Data struct {
				ReEscalateAlert struct {
					ID     string `json:"id"`
					Status string `json:"status"`
				} `json:"reEscalateAlert"`
			} `json:"data"`
			Errors []struct {
				Message string `json:"message"`
			} `json:"errors"`
		}
		require.NoError(a, json.Unmarshal(reEscalateRec.Body.Bytes(), &reEscalateResp))
		require.Empty(a, reEscalateResp.Errors)
		require.Equal(a, "TRIGGERED", reEscalateResp.Data.ReEscalateAlert.Status)

		events, err = queries.ListAuditEventsByOrganization(ctx, admin.OrganizationID)
		require.NoError(a, err)
		var reEscalateEvent *db.AuditEvent
		for i := range events {
			if events[i].Action == audit.ActionAlertReEscalated {
				reEscalateEvent = &events[i]
				break
			}
		}
		require.NotNil(a, reEscalateEvent)
	})
}
