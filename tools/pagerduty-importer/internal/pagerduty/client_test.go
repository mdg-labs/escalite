package pagerduty_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mdg-labs/escalite/tools/pagerduty-importer/internal/pagerduty"
)

func TestClientListUsers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Token token=test-token", r.Header.Get("Authorization"))
		require.Equal(t, "/users", r.URL.Path)

		_ = json.NewEncoder(w).Encode(map[string]any{
			"users": []map[string]any{
				{"id": "PU1", "name": "Alice", "email": "alice@example.com", "role": "admin"},
			},
			"more": false,
		})
	}))
	defer server.Close()

	t.Setenv("PAGERDUTY_API_BASE", server.URL)
	client := pagerduty.NewClient("test-token", server.Client())

	users, err := client.ListUsers()
	require.NoError(t, err)
	require.Len(t, users, 1)
	require.Equal(t, "alice@example.com", users[0].Email)
}

func TestClientTracksUnsupportedEscalationTargets(t *testing.T) {
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

	t.Setenv("PAGERDUTY_API_BASE", server.URL)
	client := pagerduty.NewClient("test-token", server.Client())

	_, err := client.ListEscalationPolicies()
	require.NoError(t, err)

	unsupported := client.UnsupportedObjects()
	require.Len(t, unsupported, 1)
	require.Equal(t, "escalation_target", unsupported[0].ObjectType)
	require.Equal(t, "PTNEST", unsupported[0].SourceID)
}

func TestLoadExport(t *testing.T) {
	path := t.TempDir() + "/export.json"
	data := `{
		"users": [{"id":"PU1","name":"Bob","email":"bob@example.com","role":"user"}],
		"schedules": [],
		"services": [],
		"escalation_policies": [],
		"unsupported": [{"object_type":"integration","source_id":"I1","name":"webhook","reason":"not supported"}]
	}`
	require.NoError(t, os.WriteFile(path, []byte(data), 0o600))

	reader, err := pagerduty.LoadExport(path)
	require.NoError(t, err)

	users, err := reader.ListUsers()
	require.NoError(t, err)
	require.Len(t, users, 1)

	unsupported := reader.UnsupportedObjects()
	require.Len(t, unsupported, 1)
	require.Equal(t, "integration", unsupported[0].ObjectType)
}

func TestTokenNotRequiredForExportReader(t *testing.T) {
	reader, err := pagerduty.LoadExport(writeTempExport(t))
	require.NoError(t, err)
	require.Equal(t, 0, reader.Counts().Users)
}

func writeTempExport(t *testing.T) string {
	t.Helper()
	path := t.TempDir() + "/empty.json"
	require.NoError(t, os.WriteFile(path, []byte(`{"users":[],"schedules":[],"services":[],"escalation_policies":[]}`), 0o600))
	return path
}
