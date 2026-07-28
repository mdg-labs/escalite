package handlers_test

import (
	"encoding/json"
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/allure-framework/allure-go/testify/require"

	"github.com/mdg-labs/escalite/services/api/internal/email"
	"github.com/mdg-labs/escalite/services/api/internal/handlers"
	"github.com/mdg-labs/escalite/services/api/internal/ratelimit"
	"github.com/mdg-labs/escalite/services/api/internal/server"
)

func passwordResetGraphQLTestHandler(t *testing.T, emailLimit, ipLimit int) (http.Handler, *email.RecordingSender, func()) {
	t.Helper()

	mail := &email.RecordingSender{}
	handler, _, cleanup := newTestHandlerWithOptions(t, testServerOptions{
		Mail:      mail,
		PublicURL: "http://localhost:5173",
		PasswordReset: &server.PasswordResetOptions{
			EmailLimiter: ratelimit.NewMemoryLimiter(emailLimit, time.Hour),
			IPLimiter:    ratelimit.NewMemoryLimiter(ipLimit, time.Hour),
		},
	})

	return handler, mail, cleanup
}

func TestGraphQLPasswordResetRequestSameResponseForUnknownEmail(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, mail, cleanup := passwordResetGraphQLTestHandler(t, 10, 10)
		defer cleanup()

		_ = bootstrapAdmin(t, handler)

		existing := postGraphQL(t, handler, `
		mutation {
			requestPasswordReset(input: { email: "admin@example.com" }) {
				message
			}
		}
	`, nil)
		require.Equal(a, http.StatusOK, existing.Code)
		require.Len(a, mail.Messages, 1)

		unknown := postGraphQL(t, handler, `
		mutation {
			requestPasswordReset(input: { email: "nobody@example.com" }) {
				message
			}
		}
	`, nil)
		require.Equal(a, http.StatusOK, unknown.Code)
		require.Len(a, mail.Messages, 1)

		var existingResp struct {
			Data struct {
				RequestPasswordReset struct {
					Message string `json:"message"`
				} `json:"requestPasswordReset"`
			} `json:"data"`
		}
		require.NoError(a, json.Unmarshal(existing.Body.Bytes(), &existingResp))

		var unknownResp struct {
			Data struct {
				RequestPasswordReset struct {
					Message string `json:"message"`
				} `json:"requestPasswordReset"`
			} `json:"data"`
		}
		require.NoError(a, json.Unmarshal(unknown.Body.Bytes(), &unknownResp))
		require.Equal(a, existingResp.Data.RequestPasswordReset.Message, unknownResp.Data.RequestPasswordReset.Message)
		require.Equal(a, handlers.PasswordResetAcceptedMessage, existingResp.Data.RequestPasswordReset.Message)
	})
}

func TestGraphQLPasswordResetRequestRateLimited(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, _, cleanup := passwordResetGraphQLTestHandler(t, 2, 100)
		defer cleanup()

		_ = bootstrapAdmin(t, handler)

		query := `
		mutation {
			requestPasswordReset(input: { email: "admin@example.com" }) {
				message
			}
		}
	`
		require.Equal(a, http.StatusOK, postGraphQL(t, handler, query, nil).Code)
		require.Equal(a, http.StatusOK, postGraphQL(t, handler, query, nil).Code)

		limited := postGraphQL(t, handler, query, nil)
		require.Equal(a, http.StatusOK, limited.Code)

		var resp struct {
			Errors []struct {
				Message    string `json:"message"`
				Extensions struct {
					Code string `json:"code"`
				} `json:"extensions"`
			} `json:"errors"`
		}
		require.NoError(a, json.Unmarshal(limited.Body.Bytes(), &resp))
		require.Len(a, resp.Errors, 1)
		require.Equal(a, handlers.CodeRateLimited, resp.Errors[0].Extensions.Code)
	})
}

func TestGraphQLPasswordResetConfirmUpdatesPassword(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, mail, cleanup := passwordResetGraphQLTestHandler(t, 10, 10)
		defer cleanup()

		_ = bootstrapAdmin(t, handler)

		requestRec := postGraphQL(t, handler, `
		mutation {
			requestPasswordReset(input: { email: "admin@example.com" }) {
				message
			}
		}
	`, nil)
		require.Equal(a, http.StatusOK, requestRec.Code)
		require.Len(a, mail.Messages, 1)

		token := extractResetToken(t, mail.Messages[0].Body)
		newPassword := "brand-new-password-99"

		confirmRec := postGraphQL(t, handler, `
		mutation {
			resetPassword(input: { token: "`+token+`", password: "`+newPassword+`" })
		}
	`, nil)
		require.Equal(a, http.StatusOK, confirmRec.Code)

		var confirmResp struct {
			Data struct {
				ResetPassword bool `json:"resetPassword"`
			} `json:"data"`
		}
		require.NoError(a, json.Unmarshal(confirmRec.Body.Bytes(), &confirmResp))
		require.True(a, confirmResp.Data.ResetPassword)

		loginRec := postGraphQL(t, handler, `
		mutation {
			login(input: { email: "admin@example.com", password: "`+newPassword+`" }) {
				user { id }
			}
		}
	`, nil)
		require.Equal(a, http.StatusOK, loginRec.Code)

		var loginResp struct {
			Data struct {
				Login struct {
					User struct {
						ID string `json:"id"`
					} `json:"user"`
				} `json:"login"`
			} `json:"data"`
		}
		require.NoError(a, json.Unmarshal(loginRec.Body.Bytes(), &loginResp))
		require.NotEmpty(a, loginResp.Data.Login.User.ID)
	})
}

func TestGraphQLPasswordResetConfirmRejectsExpiredToken(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, mail, cleanup := passwordResetGraphQLTestHandler(t, 10, 10)
		defer cleanup()

		_ = bootstrapAdmin(t, handler)

		requestRec := postGraphQL(t, handler, `
		mutation {
			requestPasswordReset(input: { email: "admin@example.com" }) {
				message
			}
		}
	`, nil)
		require.Equal(a, http.StatusOK, requestRec.Code)
		require.Len(a, mail.Messages, 1)

		token := extractResetToken(t, mail.Messages[0].Body)

		confirmRec := postGraphQL(t, handler, `
		mutation {
			resetPassword(input: { token: "`+token+`", password: "new-password-12345" })
		}
	`, nil)
		require.Equal(a, http.StatusOK, confirmRec.Code)

		var firstResp struct {
			Data struct {
				ResetPassword bool `json:"resetPassword"`
			} `json:"data"`
		}
		require.NoError(a, json.Unmarshal(confirmRec.Body.Bytes(), &firstResp))
		require.True(a, firstResp.Data.ResetPassword)

		reuseRec := postGraphQL(t, handler, `
		mutation {
			resetPassword(input: { token: "`+token+`", password: "another-password-99" })
		}
	`, nil)
		require.Equal(a, http.StatusOK, reuseRec.Code)

		var reuseResp struct {
			Errors []struct {
				Extensions struct {
					Code string `json:"code"`
				} `json:"extensions"`
			} `json:"errors"`
		}
		require.NoError(a, json.Unmarshal(reuseRec.Body.Bytes(), &reuseResp))
		require.Len(a, reuseResp.Errors, 1)
		require.Equal(a, handlers.CodeValidation, reuseResp.Errors[0].Extensions.Code)
	})
}

func TestGraphQLPasswordResetConfirmRejectsShortPassword(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, _, cleanup := passwordResetGraphQLTestHandler(t, 10, 10)
		defer cleanup()

		rec := postGraphQL(t, handler, `
		mutation {
			resetPassword(input: { token: "some-token", password: "short" })
		}
	`, nil)
		require.Equal(a, http.StatusOK, rec.Code)

		var resp struct {
			Errors []struct {
				Message string `json:"message"`
			} `json:"errors"`
		}
		require.NoError(a, json.Unmarshal(rec.Body.Bytes(), &resp))
		require.Len(a, resp.Errors, 1)
		require.True(a, strings.Contains(resp.Errors[0].Message, "8 characters"))
	})
}
