package handlers_test

import (
	"encoding/json"
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"testing"

	"github.com/allure-framework/allure-go/testify/require"

	"github.com/mdg-labs/escalite/services/api/internal/handlers"
)

func TestGraphQLSaveNotificationRule(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, _, cleanup := newTestHandler(t)
		defer cleanup()

		cookie := bootstrapAdmin(t, handler)

		rec := postGraphQL(t, handler, `mutation {
		saveNotificationRule(input: {
			priority: HIGH
			steps: [
				{ channel: "push", delayMinutes: 0 }
				{ channel: "email", delayMinutes: 2 }
			]
		}) {
			priority
			steps {
				channel
				delayMinutes
			}
		}
	}`, cookie)
		require.Equal(a, 200, rec.Code, rec.Body.String())

		var resp struct {
			Data struct {
				SaveNotificationRule struct {
					Priority string `json:"priority"`
					Steps    []struct {
						Channel      string `json:"channel"`
						DelayMinutes int    `json:"delayMinutes"`
					} `json:"steps"`
				} `json:"saveNotificationRule"`
			} `json:"data"`
			Errors []any `json:"errors"`
		}
		require.NoError(a, json.Unmarshal(rec.Body.Bytes(), &resp))
		require.Empty(a, resp.Errors)
		require.Equal(a, "HIGH", resp.Data.SaveNotificationRule.Priority)
		require.Equal(a, []struct {
			Channel      string `json:"channel"`
			DelayMinutes int    `json:"delayMinutes"`
		}{
			{Channel: "push", DelayMinutes: 0},
			{Channel: "email", DelayMinutes: 2},
		}, resp.Data.SaveNotificationRule.Steps)

		rec = postGraphQL(t, handler, `{
		notificationRules {
			priority
			steps {
				channel
				delayMinutes
			}
		}
	}`, cookie)
		require.Equal(a, 200, rec.Code, rec.Body.String())

		var listResp struct {
			Data struct {
				NotificationRules []struct {
					Priority string `json:"priority"`
					Steps    []struct {
						Channel      string `json:"channel"`
						DelayMinutes int    `json:"delayMinutes"`
					} `json:"steps"`
				} `json:"notificationRules"`
			} `json:"data"`
			Errors []any `json:"errors"`
		}
		require.NoError(a, json.Unmarshal(rec.Body.Bytes(), &listResp))
		require.Empty(a, listResp.Errors)
		require.Len(a, listResp.Data.NotificationRules, 1)
		require.Equal(a, "HIGH", listResp.Data.NotificationRules[0].Priority)
	})
}

func TestGraphQLSaveNotificationRuleRejectsUnknownChannel(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, _, cleanup := newTestHandler(t)
		defer cleanup()

		cookie := bootstrapAdmin(t, handler)

		rec := postGraphQL(t, handler, `mutation {
		saveNotificationRule(input: {
			priority: HIGH
			steps: [{ channel: "pagerduty", delayMinutes: 0 }]
		}) {
			id
		}
	}`, cookie)
		require.Equal(a, 200, rec.Code)

		var resp struct {
			Errors []struct {
				Message    string                 `json:"message"`
				Extensions map[string]interface{} `json:"extensions"`
			} `json:"errors"`
		}
		require.NoError(a, json.Unmarshal(rec.Body.Bytes(), &resp))
		require.NotEmpty(a, resp.Errors)
		require.Equal(a, handlers.CodeValidation, resp.Errors[0].Extensions["code"])
		require.Contains(a, resp.Errors[0].Message, `unknown notification channel "pagerduty"`)
	})
}

func TestGraphQLDeleteNotificationRule(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, _, cleanup := newTestHandler(t)
		defer cleanup()

		cookie := bootstrapAdmin(t, handler)

		rec := postGraphQL(t, handler, `mutation {
		saveNotificationRule(input: {
			priority: LOW
			steps: [{ channel: "email", delayMinutes: 0 }]
		}) {
			id
		}
	}`, cookie)
		require.Equal(a, 200, rec.Code, rec.Body.String())

		rec = postGraphQL(t, handler, `mutation {
		deleteNotificationRule(priority: LOW)
	}`, cookie)
		require.Equal(a, 200, rec.Code, rec.Body.String())

		var deleteResp struct {
			Data struct {
				DeleteNotificationRule bool `json:"deleteNotificationRule"`
			} `json:"data"`
			Errors []any `json:"errors"`
		}
		require.NoError(a, json.Unmarshal(rec.Body.Bytes(), &deleteResp))
		require.Empty(a, deleteResp.Errors)
		require.True(a, deleteResp.Data.DeleteNotificationRule)

		rec = postGraphQL(t, handler, `{ notificationRules { priority } }`, cookie)
		require.Equal(a, 200, rec.Code, rec.Body.String())

		var listResp struct {
			Data struct {
				NotificationRules []struct {
					Priority string `json:"priority"`
				} `json:"notificationRules"`
			} `json:"data"`
		}
		require.NoError(a, json.Unmarshal(rec.Body.Bytes(), &listResp))
		require.Empty(a, listResp.Data.NotificationRules)
	})
}
