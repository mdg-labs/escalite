package handlers_test

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/allure-framework/allure-go/testify/require"
	"github.com/google/uuid"

	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/engine/statuspageapi"
)

const publicStatusPageTestEncryptionKeyHex = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func TestPublicStatusPageExcludesInternalAlertFields(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, pool, cleanup := newTestHandler(t)
		defer cleanup()

		adminCookie := bootstrapAdmin(t, handler)

		queries := db.New(pool)
		admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
		require.NoError(a, err)

		team := seedTeam(t, pool, admin.OrganizationID, "Platform")

		saveRec := postGraphQL(t, handler, `mutation {
		saveStatusPage(input: {
			slug: "acme-status"
			title: "Acme Status"
			enabled: true
			frameAncestorsCsp: "https://status.example.com"
		}) {
			id
			slug
		}
	}`, adminCookie)
		require.Equal(a, http.StatusOK, saveRec.Code, saveRec.Body.String())

		componentRec := postGraphQL(t, handler, `mutation {
		createStatusPageComponent(input: {
			name: "API"
			description: "Public API"
			status: DEGRADED
			position: 1
		}) {
			id
			status
		}
	}`, adminCookie)
		require.Equal(a, http.StatusOK, componentRec.Code, componentRec.Body.String())

		var componentResp struct {
			Data struct {
				CreateStatusPageComponent struct {
					ID     string `json:"id"`
					Status string `json:"status"`
				} `json:"createStatusPageComponent"`
			} `json:"data"`
		}
		require.NoError(a, json.Unmarshal(componentRec.Body.Bytes(), &componentResp))
		require.Equal(a, "DEGRADED", componentResp.Data.CreateStatusPageComponent.Status)
		componentID := componentResp.Data.CreateStatusPageComponent.ID

		incidentRec := postGraphQL(t, handler, `mutation {
		createIncident(input: {
			teamId: "`+team.ID.String()+`"
			title: "Internal outage"
		}) {
			id
		}
	}`, adminCookie)
		require.Equal(a, http.StatusOK, incidentRec.Code, incidentRec.Body.String())

		var incidentResp struct {
			Data struct {
				CreateIncident struct {
					ID string `json:"id"`
				} `json:"createIncident"`
			} `json:"data"`
		}
		require.NoError(a, json.Unmarshal(incidentRec.Body.Bytes(), &incidentResp))
		internalIncidentID := incidentResp.Data.CreateIncident.ID

		publishRec := postGraphQL(t, handler, `mutation {
		publishIncidentToStatusPage(input: {
			incidentId: "`+internalIncidentID+`"
			affectedComponentIds: ["`+componentID+`"]
			body: "We are investigating elevated API errors."
		}) {
			id
			title
			incidentId
		}
	}`, adminCookie)
		require.Equal(a, http.StatusOK, publishRec.Code, publishRec.Body.String())

		req := httptest.NewRequest(http.MethodGet, "/api/v1/public/status/acme-status", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		require.Equal(a, http.StatusOK, rec.Code, rec.Body.String())
		require.Equal(a, "frame-ancestors https://status.example.com", rec.Header().Get("Content-Security-Policy"))

		body := rec.Body.Bytes()
		require.NotContains(a, string(body), "dedupKey")
		require.NotContains(a, string(body), "escalationState")
		require.NotContains(a, string(body), "summary")
		require.NotContains(a, string(body), internalIncidentID)
		require.NotContains(a, string(body), "serviceId")
		require.NotContains(a, string(body), "service_id")
		require.NotContains(a, string(body), "incidentId")

		var payload map[string]any
		require.NoError(a, json.Unmarshal(body, &payload))
		require.Equal(a, "Acme Status", payload["title"])
		require.Equal(a, "degraded", payload["overallStatus"])

		components, ok := payload["components"].([]any)
		require.True(a, ok)
		require.Len(a, components, 1)

		firstComponent := components[0].(map[string]any)
		require.Equal(a, componentID, firstComponent["id"])
		require.Equal(a, "API", firstComponent["name"])
		require.Equal(a, "degraded", firstComponent["status"])
		_, hasServiceID := firstComponent["serviceId"]
		require.False(a, hasServiceID)

		incidents, ok := payload["incidents"].([]any)
		require.True(a, ok)
		require.Len(a, incidents, 1)

		firstIncident := incidents[0].(map[string]any)
		require.Equal(a, "Internal outage", firstIncident["title"])
		require.Equal(a, "investigating", firstIncident["status"])
		_, hasInternalIncidentID := firstIncident["incidentId"]
		require.False(a, hasInternalIncidentID)

		subscribeBody, err := json.Marshal(map[string]string{"email": "subscriber@example.com"})
		require.NoError(a, err)

		subscribeReq := httptest.NewRequest(http.MethodPost, "/api/v1/public/status/acme-status/subscribe", bytes.NewReader(subscribeBody))
		subscribeReq.Header.Set("Content-Type", "application/json")
		subscribeRec := httptest.NewRecorder()
		handler.ServeHTTP(subscribeRec, subscribeReq)
		require.Equal(a, http.StatusOK, subscribeRec.Code, subscribeRec.Body.String())

		var subscribeResp struct {
			Subscribed bool `json:"subscribed"`
		}
		require.NoError(a, json.Unmarshal(subscribeRec.Body.Bytes(), &subscribeResp))
		require.True(a, subscribeResp.Subscribed)
	})
}

func TestPublicStatusPageComponentStatusEnumValues(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, _, cleanup := newTestHandler(t)
		defer cleanup()

		adminCookie := bootstrapAdmin(t, handler)

		saveRec := postGraphQL(t, handler, `mutation {
		saveStatusPage(input: {
			slug: "enum-check"
			title: "Enum Check"
			enabled: true
		}) {
			id
		}
	}`, adminCookie)
		require.Equal(a, http.StatusOK, saveRec.Code, saveRec.Body.String())

		statuses := []struct {
			graphql string
			public  string
		}{
			{"OPERATIONAL", "operational"},
			{"DEGRADED", "degraded"},
			{"PARTIAL_OUTAGE", "partial_outage"},
			{"MAJOR_OUTAGE", "major_outage"},
		}

		for index, status := range statuses {
			componentRec := postGraphQL(t, handler, `mutation {
			createStatusPageComponent(input: {
				name: "Component `+status.public+`"
				status: `+status.graphql+`
				position: `+strconv.Itoa(index)+`
			}) {
				status
			}
		}`, adminCookie)
			require.Equal(a, http.StatusOK, componentRec.Code, componentRec.Body.String())
		}

		req := httptest.NewRequest(http.MethodGet, "/api/v1/public/status/enum-check", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		require.Equal(a, http.StatusOK, rec.Code, rec.Body.String())

		var payload struct {
			OverallStatus string `json:"overallStatus"`
			Components    []struct {
				Status string `json:"status"`
			} `json:"components"`
		}
		require.NoError(a, json.Unmarshal(rec.Body.Bytes(), &payload))
		require.Equal(a, "major_outage", payload.OverallStatus)
		require.Len(a, payload.Components, 4)

		seen := map[string]bool{}
		for _, component := range payload.Components {
			seen[component.Status] = true
		}
		require.True(a, seen["operational"])
		require.True(a, seen["degraded"])
		require.True(a, seen["partial_outage"])
		require.True(a, seen["major_outage"])
	})
}

func TestPublicStatusPageResolvedIncidentsWindow(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, pool, cleanup := newTestHandler(t)
		defer cleanup()

		adminCookie := bootstrapAdmin(t, handler)

		queries := db.New(pool)
		admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
		require.NoError(a, err)

		team := seedTeam(t, pool, admin.OrganizationID, "Platform")

		saveRec := postGraphQL(t, handler, `mutation {
		saveStatusPage(input: {
			slug: "resolved-history"
			title: "Resolved History"
			enabled: true
		}) {
			id
		}
	}`, adminCookie)
		require.Equal(a, http.StatusOK, saveRec.Code, saveRec.Body.String())

		componentRec := postGraphQL(t, handler, `mutation {
		createStatusPageComponent(input: {
			name: "API"
			status: OPERATIONAL
			position: 0
		}) {
			id
		}
	}`, adminCookie)
		require.Equal(a, http.StatusOK, componentRec.Code, componentRec.Body.String())

		var componentResp struct {
			Data struct {
				CreateStatusPageComponent struct {
					ID string `json:"id"`
				} `json:"createStatusPageComponent"`
			} `json:"data"`
		}
		require.NoError(a, json.Unmarshal(componentRec.Body.Bytes(), &componentResp))
		componentID := componentResp.Data.CreateStatusPageComponent.ID

		incidentRec := postGraphQL(t, handler, `mutation {
		createIncident(input: {
			teamId: "`+team.ID.String()+`"
			title: "Past outage"
		}) {
			id
		}
	}`, adminCookie)
		require.Equal(a, http.StatusOK, incidentRec.Code, incidentRec.Body.String())

		var incidentResp struct {
			Data struct {
				CreateIncident struct {
					ID string `json:"id"`
				} `json:"createIncident"`
			} `json:"data"`
		}
		require.NoError(a, json.Unmarshal(incidentRec.Body.Bytes(), &incidentResp))
		internalIncidentID := incidentResp.Data.CreateIncident.ID

		publishRec := postGraphQL(t, handler, `mutation {
		publishIncidentToStatusPage(input: {
			incidentId: "`+internalIncidentID+`"
			affectedComponentIds: ["`+componentID+`"]
			body: "We are investigating elevated API errors."
		}) {
			id
			title
		}
	}`, adminCookie)
		require.Equal(a, http.StatusOK, publishRec.Code, publishRec.Body.String())

		var publishResp struct {
			Data struct {
				PublishIncidentToStatusPage struct {
					ID    string `json:"id"`
					Title string `json:"title"`
				} `json:"publishIncidentToStatusPage"`
			} `json:"data"`
		}
		require.NoError(a, json.Unmarshal(publishRec.Body.Bytes(), &publishResp))
		statusPageIncidentID := publishResp.Data.PublishIncidentToStatusPage.ID

		activeReq := httptest.NewRequest(http.MethodGet, "/api/v1/public/status/resolved-history", nil)
		activeRec := httptest.NewRecorder()
		handler.ServeHTTP(activeRec, activeReq)
		require.Equal(a, http.StatusOK, activeRec.Code, activeRec.Body.String())

		var activePayload struct {
			Incidents         []map[string]any `json:"incidents"`
			ResolvedIncidents []map[string]any `json:"resolvedIncidents"`
		}
		require.NoError(a, json.Unmarshal(activeRec.Body.Bytes(), &activePayload))
		require.Len(a, activePayload.Incidents, 1)
		require.Empty(a, activePayload.ResolvedIncidents)

		resolveRec := postGraphQL(t, handler, `mutation {
		updateStatusPageIncidentStatus(input: {
			id: "`+statusPageIncidentID+`"
			status: RESOLVED
			body: "The incident has been resolved."
		}) {
			id
			status
		}
	}`, adminCookie)
		require.Equal(a, http.StatusOK, resolveRec.Code, resolveRec.Body.String())

		resolvedReq := httptest.NewRequest(http.MethodGet, "/api/v1/public/status/resolved-history", nil)
		resolvedRec := httptest.NewRecorder()
		handler.ServeHTTP(resolvedRec, resolvedReq)
		require.Equal(a, http.StatusOK, resolvedRec.Code, resolvedRec.Body.String())

		var resolvedPayload struct {
			Incidents         []map[string]any `json:"incidents"`
			ResolvedIncidents []map[string]any `json:"resolvedIncidents"`
		}
		require.NoError(a, json.Unmarshal(resolvedRec.Body.Bytes(), &resolvedPayload))
		require.Empty(a, resolvedPayload.Incidents)
		require.Len(a, resolvedPayload.ResolvedIncidents, 1)

		firstResolved := resolvedPayload.ResolvedIncidents[0]
		require.Equal(a, "Past outage", firstResolved["title"])
		require.Equal(a, "resolved", firstResolved["status"])
		require.NotNil(a, firstResolved["resolvedAt"])
	})
}

func TestPublicStatusPageUnsubscribeTokenIsIdempotent(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, pool, cleanup := newTestHandler(t)
		defer cleanup()

		adminCookie := bootstrapAdmin(t, handler)

		saveRec := postGraphQL(t, handler, `mutation {
		saveStatusPage(input: {
			slug: "unsubscribe-status"
			title: "Unsubscribe Status"
			enabled: true
		}) {
			id
		}
	}`, adminCookie)
		require.Equal(a, http.StatusOK, saveRec.Code, saveRec.Body.String())

		subscribeBody, err := json.Marshal(map[string]string{"email": "subscriber@example.com"})
		require.NoError(a, err)

		subscribeReq := httptest.NewRequest(http.MethodPost, "/api/v1/public/status/unsubscribe-status/subscribe", bytes.NewReader(subscribeBody))
		subscribeReq.Header.Set("Content-Type", "application/json")
		subscribeRec := httptest.NewRecorder()
		handler.ServeHTTP(subscribeRec, subscribeReq)
		require.Equal(a, http.StatusOK, subscribeRec.Code, subscribeRec.Body.String())

		queries := db.New(pool)
		admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
		require.NoError(a, err)

		page, err := queries.GetStatusPageByOrganizationID(context.Background(), admin.OrganizationID)
		require.NoError(a, err)

		subscriptions, err := queries.ListStatusPageSubscriptions(context.Background(), db.ListStatusPageSubscriptionsParams{
			StatusPageID:   page.ID,
			OrganizationID: admin.OrganizationID,
		})
		require.NoError(a, err)
		require.Len(a, subscriptions, 1)

		signingKey, err := hex.DecodeString(publicStatusPageTestEncryptionKeyHex)
		require.NoError(a, err)

		token, err := statuspageapi.SignUnsubscribeToken(signingKey, statuspageapi.UnsubscribeClaims{
			SubscriptionID: subscriptions[0].ID,
			OrganizationID: admin.OrganizationID,
		})
		require.NoError(a, err)

		unsubscribeBody, err := json.Marshal(map[string]string{"token": token})
		require.NoError(a, err)

		for range 2 {
			unsubscribeReq := httptest.NewRequest(http.MethodPost, "/api/v1/public/status/unsubscribe", bytes.NewReader(unsubscribeBody))
			unsubscribeReq.Header.Set("Content-Type", "application/json")
			unsubscribeRec := httptest.NewRecorder()
			handler.ServeHTTP(unsubscribeRec, unsubscribeReq)
			require.Equal(a, http.StatusOK, unsubscribeRec.Code, unsubscribeRec.Body.String())

			var unsubscribeResp struct {
				Unsubscribed bool `json:"unsubscribed"`
			}
			require.NoError(a, json.Unmarshal(unsubscribeRec.Body.Bytes(), &unsubscribeResp))
			require.True(a, unsubscribeResp.Unsubscribed)
		}

		updated, err := queries.UnsubscribeStatusPageSubscription(context.Background(), db.UnsubscribeStatusPageSubscriptionParams{
			ID:             subscriptions[0].ID,
			OrganizationID: admin.OrganizationID,
		})
		require.NoError(a, err)
		require.True(a, updated.UnsubscribedAt.Valid)

		active, err := queries.ListStatusPageSubscriptions(context.Background(), db.ListStatusPageSubscriptionsParams{
			StatusPageID:   page.ID,
			OrganizationID: admin.OrganizationID,
		})
		require.NoError(a, err)
		require.Empty(a, active)

		invalidToken, err := statuspageapi.SignUnsubscribeToken(signingKey, statuspageapi.UnsubscribeClaims{
			SubscriptionID: uuid.Must(uuid.NewV7()),
			OrganizationID: admin.OrganizationID,
		})
		require.NoError(a, err)

		invalidBody, err := json.Marshal(map[string]string{"token": invalidToken})
		require.NoError(a, err)

		invalidReq := httptest.NewRequest(http.MethodPost, "/api/v1/public/status/unsubscribe", bytes.NewReader(invalidBody))
		invalidReq.Header.Set("Content-Type", "application/json")
		invalidRec := httptest.NewRecorder()
		handler.ServeHTTP(invalidRec, invalidReq)
		require.Equal(a, http.StatusBadRequest, invalidRec.Code, invalidRec.Body.String())
	})
}
