package handlers_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"

	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/api/internal/handlers"
)

func seedTriggeredAlert(t *testing.T, pool db.DBTX, orgID, serviceID uuid.UUID, dedupKey string) db.Alert {
	t.Helper()

	queries := db.New(pool)
	alert, err := queries.CreateTriggeredAlert(context.Background(), db.CreateTriggeredAlertParams{
		ID:              uuid.Must(uuid.NewV7()),
		OrganizationID:  orgID,
		ServiceID:       serviceID,
		IntegrationKeyID: pgtype.UUID{},
		DedupKey:        dedupKey,
		Summary:         "Memory above threshold",
		Description:     pgtype.Text{String: "usage at 95%", Valid: true},
		Priority:        "high",
		EscalationState: []byte(`{"current_step":1}`),
	})
	require.NoError(t, err)
	return alert
}

func TestGraphQLAcknowledgeAndCloseAlert(t *testing.T) {
	handler, pool, cleanup := newTestHandler(t)
	defer cleanup()

	adminCookie := bootstrapAdmin(t, handler)

	queries := db.New(pool)
	admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
	require.NoError(t, err)

	team := seedTeam(t, pool, admin.OrganizationID, "Platform")
	service := seedService(t, pool, admin.OrganizationID, team.ID, "checkout-api")
	alert := seedTriggeredAlert(t, pool, admin.OrganizationID, service.ID, "memory-high")

	ackRec := postGraphQL(t, handler, `mutation {
		acknowledgeAlert(id: "`+alert.ID.String()+`") {
			id
			status
			acknowledgedAt
			acknowledgedBy { id email }
		}
	}`, adminCookie)
	require.Equal(t, 200, ackRec.Code)

	var ackResp struct {
		Data struct {
			AcknowledgeAlert struct {
				ID             string  `json:"id"`
				Status         string  `json:"status"`
				AcknowledgedAt *string `json:"acknowledgedAt"`
				AcknowledgedBy struct {
					ID    string `json:"id"`
					Email string `json:"email"`
				} `json:"acknowledgedBy"`
			} `json:"acknowledgeAlert"`
		} `json:"data"`
		Errors []struct {
			Message    string                 `json:"message"`
			Extensions map[string]interface{} `json:"extensions"`
		} `json:"errors"`
	}
	require.NoError(t, json.Unmarshal(ackRec.Body.Bytes(), &ackResp))
	require.Empty(t, ackResp.Errors)
	require.Equal(t, alert.ID.String(), ackResp.Data.AcknowledgeAlert.ID)
	require.Equal(t, "ACKNOWLEDGED", ackResp.Data.AcknowledgeAlert.Status)
	require.NotNil(t, ackResp.Data.AcknowledgeAlert.AcknowledgedAt)
	require.Equal(t, admin.ID.String(), ackResp.Data.AcknowledgeAlert.AcknowledgedBy.ID)
	require.Equal(t, "admin@example.com", ackResp.Data.AcknowledgeAlert.AcknowledgedBy.Email)

	closeRec := postGraphQL(t, handler, `mutation {
		closeAlert(id: "`+alert.ID.String()+`") {
			id
			status
			closedAt
		}
	}`, adminCookie)
	require.Equal(t, 200, closeRec.Code)

	var closeResp struct {
		Data struct {
			CloseAlert struct {
				ID       string  `json:"id"`
				Status   string  `json:"status"`
				ClosedAt *string `json:"closedAt"`
			} `json:"closeAlert"`
		} `json:"data"`
		Errors []struct {
			Message    string                 `json:"message"`
			Extensions map[string]interface{} `json:"extensions"`
		} `json:"errors"`
	}
	require.NoError(t, json.Unmarshal(closeRec.Body.Bytes(), &closeResp))
	require.Empty(t, closeResp.Errors)
	require.Equal(t, "CLOSED", closeResp.Data.CloseAlert.Status)
	require.NotNil(t, closeResp.Data.CloseAlert.ClosedAt)

	reAckRec := postGraphQL(t, handler, `mutation {
		acknowledgeAlert(id: "`+alert.ID.String()+`") {
			id
		}
	}`, adminCookie)
	require.Equal(t, 200, reAckRec.Code)

	var reAckResp struct {
		Errors []struct {
			Message    string                 `json:"message"`
			Extensions map[string]interface{} `json:"extensions"`
		} `json:"errors"`
	}
	require.NoError(t, json.Unmarshal(reAckRec.Body.Bytes(), &reAckResp))
	require.Len(t, reAckResp.Errors, 1)
	require.Equal(t, handlers.CodeValidation, reAckResp.Errors[0].Extensions["code"])
}

func TestGraphQLAcknowledgeAlertRequiresAuth(t *testing.T) {
	handler, pool, cleanup := newTestHandler(t)
	defer cleanup()

	bootstrapAdmin(t, handler)

	queries := db.New(pool)
	admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
	require.NoError(t, err)

	team := seedTeam(t, pool, admin.OrganizationID, "Platform")
	service := seedService(t, pool, admin.OrganizationID, team.ID, "checkout-api")
	alert := seedTriggeredAlert(t, pool, admin.OrganizationID, service.ID, "disk-full")

	rec := postGraphQL(t, handler, `mutation {
		acknowledgeAlert(id: "`+alert.ID.String()+`") { id }
	}`, nil)
	require.Equal(t, 200, rec.Code)

	var resp struct {
		Errors []struct {
			Extensions map[string]interface{} `json:"extensions"`
		} `json:"errors"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Len(t, resp.Errors, 1)
	require.Equal(t, handlers.CodeUnauthenticated, resp.Errors[0].Extensions["code"])
}
