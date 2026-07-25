package handlers_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mdg-labs/escalite/services/api/internal/handlers"
)

func TestGraphQLSaveNotificationRule(t *testing.T) {
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
	require.Equal(t, 200, rec.Code, rec.Body.String())

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
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Empty(t, resp.Errors)
	require.Equal(t, "HIGH", resp.Data.SaveNotificationRule.Priority)
	require.Equal(t, []struct {
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
	require.Equal(t, 200, rec.Code, rec.Body.String())

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
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &listResp))
	require.Empty(t, listResp.Errors)
	require.Len(t, listResp.Data.NotificationRules, 1)
	require.Equal(t, "HIGH", listResp.Data.NotificationRules[0].Priority)
}

func TestGraphQLSaveNotificationRuleRejectsUnknownChannel(t *testing.T) {
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
	require.Equal(t, 200, rec.Code)

	var resp struct {
		Errors []struct {
			Message    string                 `json:"message"`
			Extensions map[string]interface{} `json:"extensions"`
		} `json:"errors"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.NotEmpty(t, resp.Errors)
	require.Equal(t, handlers.CodeValidation, resp.Errors[0].Extensions["code"])
	require.Contains(t, resp.Errors[0].Message, `unknown notification channel "pagerduty"`)
}

func TestGraphQLDeleteNotificationRule(t *testing.T) {
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
	require.Equal(t, 200, rec.Code, rec.Body.String())

	rec = postGraphQL(t, handler, `mutation {
		deleteNotificationRule(priority: LOW)
	}`, cookie)
	require.Equal(t, 200, rec.Code, rec.Body.String())

	var deleteResp struct {
		Data struct {
			DeleteNotificationRule bool `json:"deleteNotificationRule"`
		} `json:"data"`
		Errors []any `json:"errors"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &deleteResp))
	require.Empty(t, deleteResp.Errors)
	require.True(t, deleteResp.Data.DeleteNotificationRule)

	rec = postGraphQL(t, handler, `{ notificationRules { priority } }`, cookie)
	require.Equal(t, 200, rec.Code, rec.Body.String())

	var listResp struct {
		Data struct {
			NotificationRules []struct {
				Priority string `json:"priority"`
			} `json:"notificationRules"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &listResp))
	require.Empty(t, listResp.Data.NotificationRules)
}
