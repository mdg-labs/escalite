package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mdg-labs/escalite/services/api/internal/db"
)

func TestPublicStatusPageExcludesInternalAlertFields(t *testing.T) {
	handler, pool, cleanup := newTestHandler(t)
	defer cleanup()

	adminCookie := bootstrapAdmin(t, handler)

	queries := db.New(pool)
	admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
	require.NoError(t, err)

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
	require.Equal(t, http.StatusOK, saveRec.Code, saveRec.Body.String())

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
	require.Equal(t, http.StatusOK, componentRec.Code, componentRec.Body.String())

	var componentResp struct {
		Data struct {
			CreateStatusPageComponent struct {
				ID     string `json:"id"`
				Status string `json:"status"`
			} `json:"createStatusPageComponent"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(componentRec.Body.Bytes(), &componentResp))
	require.Equal(t, "DEGRADED", componentResp.Data.CreateStatusPageComponent.Status)
	componentID := componentResp.Data.CreateStatusPageComponent.ID

	incidentRec := postGraphQL(t, handler, `mutation {
		createIncident(input: {
			teamId: "`+team.ID.String()+`"
			title: "Internal outage"
		}) {
			id
		}
	}`, adminCookie)
	require.Equal(t, http.StatusOK, incidentRec.Code, incidentRec.Body.String())

	var incidentResp struct {
		Data struct {
			CreateIncident struct {
				ID string `json:"id"`
			} `json:"createIncident"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(incidentRec.Body.Bytes(), &incidentResp))
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
	require.Equal(t, http.StatusOK, publishRec.Code, publishRec.Body.String())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/public/status/acme-status", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.Equal(t, "frame-ancestors https://status.example.com", rec.Header().Get("Content-Security-Policy"))

	body := rec.Body.Bytes()
	require.NotContains(t, string(body), "dedupKey")
	require.NotContains(t, string(body), "escalationState")
	require.NotContains(t, string(body), "summary")
	require.NotContains(t, string(body), internalIncidentID)
	require.NotContains(t, string(body), "serviceId")
	require.NotContains(t, string(body), "service_id")
	require.NotContains(t, string(body), "incidentId")

	var payload map[string]any
	require.NoError(t, json.Unmarshal(body, &payload))
	require.Equal(t, "Acme Status", payload["title"])
	require.Equal(t, "degraded", payload["overallStatus"])

	components, ok := payload["components"].([]any)
	require.True(t, ok)
	require.Len(t, components, 1)

	firstComponent := components[0].(map[string]any)
	require.Equal(t, componentID, firstComponent["id"])
	require.Equal(t, "API", firstComponent["name"])
	require.Equal(t, "degraded", firstComponent["status"])
	_, hasServiceID := firstComponent["serviceId"]
	require.False(t, hasServiceID)

	incidents, ok := payload["incidents"].([]any)
	require.True(t, ok)
	require.Len(t, incidents, 1)

	firstIncident := incidents[0].(map[string]any)
	require.Equal(t, "Internal outage", firstIncident["title"])
	require.Equal(t, "investigating", firstIncident["status"])
	_, hasInternalIncidentID := firstIncident["incidentId"]
	require.False(t, hasInternalIncidentID)

	subscribeBody, err := json.Marshal(map[string]string{"email": "subscriber@example.com"})
	require.NoError(t, err)

	subscribeReq := httptest.NewRequest(http.MethodPost, "/api/v1/public/status/acme-status/subscribe", bytes.NewReader(subscribeBody))
	subscribeReq.Header.Set("Content-Type", "application/json")
	subscribeRec := httptest.NewRecorder()
	handler.ServeHTTP(subscribeRec, subscribeReq)
	require.Equal(t, http.StatusOK, subscribeRec.Code, subscribeRec.Body.String())

	var subscribeResp struct {
		Subscribed bool `json:"subscribed"`
	}
	require.NoError(t, json.Unmarshal(subscribeRec.Body.Bytes(), &subscribeResp))
	require.True(t, subscribeResp.Subscribed)
}

func TestPublicStatusPageComponentStatusEnumValues(t *testing.T) {
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
	require.Equal(t, http.StatusOK, saveRec.Code, saveRec.Body.String())

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
		require.Equal(t, http.StatusOK, componentRec.Code, componentRec.Body.String())
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/public/status/enum-check", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var payload struct {
		OverallStatus string `json:"overallStatus"`
		Components    []struct {
			Status string `json:"status"`
		} `json:"components"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	require.Equal(t, "major_outage", payload.OverallStatus)
	require.Len(t, payload.Components, 4)

	seen := map[string]bool{}
	for _, component := range payload.Components {
		seen[component.Status] = true
	}
	require.True(t, seen["operational"])
	require.True(t, seen["degraded"])
	require.True(t, seen["partial_outage"])
	require.True(t, seen["major_outage"])
}

func TestPublicStatusPageResolvedIncidentsWindow(t *testing.T) {
	handler, pool, cleanup := newTestHandler(t)
	defer cleanup()

	adminCookie := bootstrapAdmin(t, handler)

	queries := db.New(pool)
	admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
	require.NoError(t, err)

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
	require.Equal(t, http.StatusOK, saveRec.Code, saveRec.Body.String())

	componentRec := postGraphQL(t, handler, `mutation {
		createStatusPageComponent(input: {
			name: "API"
			status: OPERATIONAL
			position: 0
		}) {
			id
		}
	}`, adminCookie)
	require.Equal(t, http.StatusOK, componentRec.Code, componentRec.Body.String())

	var componentResp struct {
		Data struct {
			CreateStatusPageComponent struct {
				ID string `json:"id"`
			} `json:"createStatusPageComponent"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(componentRec.Body.Bytes(), &componentResp))
	componentID := componentResp.Data.CreateStatusPageComponent.ID

	incidentRec := postGraphQL(t, handler, `mutation {
		createIncident(input: {
			teamId: "`+team.ID.String()+`"
			title: "Past outage"
		}) {
			id
		}
	}`, adminCookie)
	require.Equal(t, http.StatusOK, incidentRec.Code, incidentRec.Body.String())

	var incidentResp struct {
		Data struct {
			CreateIncident struct {
				ID string `json:"id"`
			} `json:"createIncident"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(incidentRec.Body.Bytes(), &incidentResp))
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
	require.Equal(t, http.StatusOK, publishRec.Code, publishRec.Body.String())

	var publishResp struct {
		Data struct {
			PublishIncidentToStatusPage struct {
				ID    string `json:"id"`
				Title string `json:"title"`
			} `json:"publishIncidentToStatusPage"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(publishRec.Body.Bytes(), &publishResp))
	statusPageIncidentID := publishResp.Data.PublishIncidentToStatusPage.ID

	activeReq := httptest.NewRequest(http.MethodGet, "/api/v1/public/status/resolved-history", nil)
	activeRec := httptest.NewRecorder()
	handler.ServeHTTP(activeRec, activeReq)
	require.Equal(t, http.StatusOK, activeRec.Code, activeRec.Body.String())

	var activePayload struct {
		Incidents         []map[string]any `json:"incidents"`
		ResolvedIncidents []map[string]any `json:"resolvedIncidents"`
	}
	require.NoError(t, json.Unmarshal(activeRec.Body.Bytes(), &activePayload))
	require.Len(t, activePayload.Incidents, 1)
	require.Empty(t, activePayload.ResolvedIncidents)

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
	require.Equal(t, http.StatusOK, resolveRec.Code, resolveRec.Body.String())

	resolvedReq := httptest.NewRequest(http.MethodGet, "/api/v1/public/status/resolved-history", nil)
	resolvedRec := httptest.NewRecorder()
	handler.ServeHTTP(resolvedRec, resolvedReq)
	require.Equal(t, http.StatusOK, resolvedRec.Code, resolvedRec.Body.String())

	var resolvedPayload struct {
		Incidents         []map[string]any `json:"incidents"`
		ResolvedIncidents []map[string]any `json:"resolvedIncidents"`
	}
	require.NoError(t, json.Unmarshal(resolvedRec.Body.Bytes(), &resolvedPayload))
	require.Empty(t, resolvedPayload.Incidents)
	require.Len(t, resolvedPayload.ResolvedIncidents, 1)

	firstResolved := resolvedPayload.ResolvedIncidents[0]
	require.Equal(t, "Past outage", firstResolved["title"])
	require.Equal(t, "resolved", firstResolved["status"])
	require.NotNil(t, firstResolved["resolvedAt"])
}
