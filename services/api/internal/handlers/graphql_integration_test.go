package handlers_test

import (
	"bytes"
	"encoding/json"
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/allure-framework/allure-go/testify/require"

	"github.com/mdg-labs/escalite/services/api/internal/auth"
	"github.com/mdg-labs/escalite/services/api/internal/graphql"
	"github.com/mdg-labs/escalite/services/api/internal/handlers"
	"github.com/mdg-labs/escalite/services/api/internal/server"
)

func postGraphQL(t *testing.T, handler http.Handler, query string, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()

	body, err := json.Marshal(map[string]string{"query": query})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/graphql", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if cookie != nil {
		req.AddCookie(cookie)
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func postGraphQLWithBearer(t *testing.T, handler http.Handler, query string, bearerToken string) *httptest.ResponseRecorder {
	t.Helper()

	body, err := json.Marshal(map[string]string{"query": query})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/graphql", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+bearerToken)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func postMobileAuthCode(t *testing.T, handler http.Handler, sessionCookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/mobile/auth/code", nil)
	req.AddCookie(sessionCookie)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func postMobileAuthExchange(t *testing.T, handler http.Handler, code string) *httptest.ResponseRecorder {
	t.Helper()

	payload, err := json.Marshal(map[string]string{"code": code})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/mobile/auth/exchange", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func TestGraphQLHealthQuery(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, _, cleanup := newTestHandler(t)
		defer cleanup()

		rec := postGraphQL(t, handler, `{ health { status } }`, nil)
		require.Equal(a, http.StatusOK, rec.Code)

		var resp struct {
			Data struct {
				Health struct {
					Status string `json:"status"`
				} `json:"health"`
			} `json:"data"`
		}
		require.NoError(a, json.Unmarshal(rec.Body.Bytes(), &resp))
		require.Equal(a, "ok", resp.Data.Health.Status)
	})
}

func TestGraphQLMeWithoutSessionReturnsNull(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, _, cleanup := newTestHandler(t)
		defer cleanup()

		rec := postGraphQL(t, handler, `{ me { id email } }`, nil)
		require.Equal(a, http.StatusOK, rec.Code)

		var resp struct {
			Data struct {
				Me *struct {
					ID string `json:"id"`
				} `json:"me"`
			} `json:"data"`
		}
		require.NoError(a, json.Unmarshal(rec.Body.Bytes(), &resp))
		require.Nil(a, resp.Data.Me)
	})
}

func TestGraphQLMeWithSession(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, _, cleanup := newTestHandler(t)
		defer cleanup()

		cookie := bootstrapAdmin(t, handler)

		rec := postGraphQL(t, handler, `{ me { email role } }`, cookie)
		require.Equal(a, http.StatusOK, rec.Code)

		var resp struct {
			Data struct {
				Me struct {
					Email string `json:"email"`
					Role  string `json:"role"`
				} `json:"me"`
			} `json:"data"`
		}
		require.NoError(a, json.Unmarshal(rec.Body.Bytes(), &resp))
		require.Equal(a, "admin@example.com", resp.Data.Me.Email)
		require.Equal(a, "ADMIN", resp.Data.Me.Role)
	})
}

func TestGraphQLLoginSetsSessionCookie(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, _, cleanup := newTestHandler(t)
		defer cleanup()

		_ = bootstrapAdmin(t, handler)

		rec := postGraphQL(t, handler, `mutation {
		login(input: { email: "admin@example.com", password: "correct-horse-battery-staple" }) {
			user { email role }
		}
	}`, nil)
		require.Equal(a, http.StatusOK, rec.Code)

		var resp struct {
			Data struct {
				Login struct {
					User struct {
						Email string `json:"email"`
						Role  string `json:"role"`
					} `json:"user"`
				} `json:"login"`
			} `json:"data"`
		}
		require.NoError(a, json.Unmarshal(rec.Body.Bytes(), &resp))
		require.Equal(a, "admin@example.com", resp.Data.Login.User.Email)
		require.Equal(a, "ADMIN", resp.Data.Login.User.Role)

		sessionCookie := findSessionCookie(t, rec)
		require.NotEmpty(a, sessionCookie.Value)
	})
}

func TestGraphQLErrorIncludesExtensionsCode(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, _, cleanup := newTestHandler(t)
		defer cleanup()

		rec := postGraphQL(t, handler, `mutation {
		login(input: { email: "admin@example.com", password: "wrong-password" }) {
			user { email }
		}
	}`, nil)
		require.Equal(a, http.StatusOK, rec.Code)

		var resp struct {
			Errors []struct {
				Message    string                 `json:"message"`
				Extensions map[string]interface{} `json:"extensions"`
			} `json:"errors"`
		}
		require.NoError(a, json.Unmarshal(rec.Body.Bytes(), &resp))
		require.Len(a, resp.Errors, 1)
		require.Equal(a, handlers.CodeUnauthenticated, resp.Errors[0].Extensions["code"])
	})
}

func TestGraphQLProductionIntrospectionRequiresAuth(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		_, pool, cleanup := newTestHandlerWithOptions(t, testServerOptions{})
		defer cleanup()

		router := server.New(server.Dependencies{
			Logger: slog.Default(),
			Pool:   pool,
			GraphQL: graphql.Options{
				Production:    true,
				MaxDepth:      15,
				MaxComplexity: 100,
			},
		})

		rec := postGraphQL(t, router, `{ __schema { queryType { name } } }`, nil)
		require.Equal(a, http.StatusOK, rec.Code)

		var unauthResp struct {
			Errors []struct {
				Extensions map[string]interface{} `json:"extensions"`
			} `json:"errors"`
		}
		require.NoError(a, json.Unmarshal(rec.Body.Bytes(), &unauthResp))
		require.NotEmpty(a, unauthResp.Errors)

		cookie := bootstrapAdmin(t, router)
		rec = postGraphQL(t, router, `{ __schema { queryType { name } } }`, cookie)
		require.Equal(a, http.StatusOK, rec.Code)

		var authResp struct {
			Data struct {
				Schema struct {
					QueryType struct {
						Name string `json:"name"`
					} `json:"queryType"`
				} `json:"__schema"`
			} `json:"data"`
		}
		require.NoError(a, json.Unmarshal(rec.Body.Bytes(), &authResp))
		require.Equal(a, "Query", authResp.Data.Schema.QueryType.Name)
	})
}

func TestGraphQLDepthLimitIncludesValidationCode(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		_, pool, cleanup := newTestHandlerWithOptions(t, testServerOptions{})
		defer cleanup()

		router := server.New(server.Dependencies{
			Logger: slog.Default(),
			Pool:   pool,
			GraphQL: graphql.Options{
				MaxDepth:      1,
				MaxComplexity: 100,
			},
		})

		rec := postGraphQL(t, router, `{ health { status } }`, nil)
		require.Equal(a, http.StatusOK, rec.Code)

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

func TestGraphQLSetupMutation(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, _, cleanup := newTestHandler(t)
		defer cleanup()

		rec := postGraphQL(t, handler, `mutation {
		setup(input: {
			organizationName: "GraphQL Org"
			email: "admin@example.com"
			password: "correct-horse-battery-staple"
		}) {
			organization { name }
			user { email role }
		}
	}`, nil)
		require.Equal(a, http.StatusOK, rec.Code)

		var resp struct {
			Data struct {
				Setup struct {
					Organization struct {
						Name string `json:"name"`
					} `json:"organization"`
					User struct {
						Email string `json:"email"`
						Role  string `json:"role"`
					} `json:"user"`
				} `json:"setup"`
			} `json:"data"`
		}
		require.NoError(a, json.Unmarshal(rec.Body.Bytes(), &resp))
		require.Equal(a, "GraphQL Org", resp.Data.Setup.Organization.Name)
		require.Equal(a, "admin@example.com", resp.Data.Setup.User.Email)
		require.Equal(a, "ADMIN", resp.Data.Setup.User.Role)

		sessionCookie := findSessionCookie(t, rec)
		require.Equal(a, auth.SessionCookieName, sessionCookie.Name)
	})
}

func TestIncidentRoleDefinitionsSeededAfterSetup(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, _, cleanup := newTestHandler(t)
		defer cleanup()

		cookie := bootstrapAdmin(t, handler)

		rec := postGraphQL(t, handler, `{
		incidentRoleDefinitions {
			name
			sortOrder
		}
	}`, cookie)
		require.Equal(a, http.StatusOK, rec.Code)

		var resp struct {
			Data struct {
				IncidentRoleDefinitions []struct {
					Name      string `json:"name"`
					SortOrder int    `json:"sortOrder"`
				} `json:"incidentRoleDefinitions"`
			} `json:"data"`
		}
		require.NoError(a, json.Unmarshal(rec.Body.Bytes(), &resp))
		require.Len(a, resp.Data.IncidentRoleDefinitions, 2)
		require.Equal(a, "IC", resp.Data.IncidentRoleDefinitions[0].Name)
		require.Equal(a, 0, resp.Data.IncidentRoleDefinitions[0].SortOrder)
		require.Equal(a, "Comms Lead", resp.Data.IncidentRoleDefinitions[1].Name)
		require.Equal(a, 1, resp.Data.IncidentRoleDefinitions[1].SortOrder)
	})
}
