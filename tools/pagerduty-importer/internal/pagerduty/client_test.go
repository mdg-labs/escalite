package pagerduty_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	allure "github.com/allure-framework/allure-go/commons/gotest"
	"github.com/allure-framework/allure-go/testify/require"

	"github.com/mdg-labs/escalite/tools/pagerduty-importer/internal/pagerduty"
)

func TestClientListUsers(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(a.T(), "Token token=test-token", r.Header.Get("Authorization"))
			require.Equal(a.T(), "/users", r.URL.Path)

			_ = json.NewEncoder(w).Encode(map[string]any{
				"users": []map[string]any{
					{"id": "PU1", "name": "Alice", "email": "alice@example.com", "role": "admin"},
				},
				"more": false,
			})
		}))
		defer server.Close()

		a.T().Setenv("PAGERDUTY_API_BASE", server.URL)
		client := pagerduty.NewClient("test-token", server.Client())

		users, err := client.ListUsers()
		require.NoError(a, err)
		require.Len(a, users, 1)
		require.Equal(a, "alice@example.com", users[0].Email)
	})
}

func TestClientTracksUnsupportedEscalationTargets(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/users":
				_ = json.NewEncoder(w).Encode(map[string]any{"users": []any{}, "more": false})
			case "/schedules":
				_ = json.NewEncoder(w).Encode(map[string]any{"schedules": []any{}, "more": false})
			case "/services":
				_ = json.NewEncoder(w).Encode(map[string]any{"services": []any{}, "more": false})
			case "/escalation_policies":
				_ = json.NewEncoder(w).Encode(map[string]any{
					"escalation_policies": []map[string]any{
						{
							"id": "PP1", "name": "Default", "num_loops": 0,
							"escalation_rules": []map[string]any{
								{
									"id": "PR1", "escalation_delay_in_minutes": 10,
									"targets": []map[string]any{
										{"id": "PTNEST", "type": "escalation_policy_reference"},
										{"id": "PU1", "type": "user_reference"},
									},
								},
							},
						},
					},
					"more": false,
				})
			default:
				http.NotFound(w, r)
			}
		}))
		defer server.Close()

		a.T().Setenv("PAGERDUTY_API_BASE", server.URL)
		client := pagerduty.NewClient("test-token", server.Client())

		_, err := client.ListEscalationPolicies()
		require.NoError(a, err)

		unsupported := client.UnsupportedObjects()
		require.Len(a, unsupported, 1)
		require.Equal(a, "escalation_target", unsupported[0].ObjectType)
		require.Equal(a, "PTNEST", unsupported[0].SourceID)
	})
}

func TestLoadExport(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		path := a.T().TempDir() + "/export.json"
		data := `{
		"users": [{"id":"PU1","name":"Bob","email":"bob@example.com","role":"user"}],
		"schedules": [],
		"services": [],
		"escalation_policies": [],
		"unsupported": [{"object_type":"integration","source_id":"I1","name":"webhook","reason":"not supported"}]
	}`
		require.NoError(a, os.WriteFile(path, []byte(data), 0o600))

		reader, err := pagerduty.LoadExport(path)
		require.NoError(a, err)

		users, err := reader.ListUsers()
		require.NoError(a, err)
		require.Len(a, users, 1)

		unsupported := reader.UnsupportedObjects()
		require.Len(a, unsupported, 1)
		require.Equal(a, "integration", unsupported[0].ObjectType)
	})
}

func TestTokenNotRequiredForExportReader(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		reader, err := pagerduty.LoadExport(writeTempExport(a.T()))
		require.NoError(a, err)
		require.Equal(a, 0, reader.Counts().Users)
	})
}

func writeTempExport(t *testing.T) string {
	t.Helper()
	path := t.TempDir() + "/empty.json"
	require.NoError(t, os.WriteFile(path, []byte(`{"users":[],"schedules":[],"services":[],"escalation_policies":[]}`), 0o600))
	return path
}
