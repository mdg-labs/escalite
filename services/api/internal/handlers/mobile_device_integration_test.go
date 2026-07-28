package handlers_test

import (
	"encoding/json"
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"net/http"
	"testing"

	"github.com/allure-framework/allure-go/testify/require"
)

func TestGraphQLRegisterMobileDeviceUpsertsDuplicateToken(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, _, cleanup := newTestHandler(t)
		defer cleanup()

		refreshToken := bootstrapMobileRefreshToken(t, handler, bootstrapAdmin(t, handler))
		token := "ExponentPushToken[duplicate-token]"

		firstRec := postGraphQLWithBearer(t, handler, `mutation {
		registerMobileDevice(input: {
			expoPushToken: "`+token+`"
			platform: "ios"
			deviceLabel: "iPhone"
		}) {
			id
			pushTokenPrefix
			platform
			deviceLabel
			revokedAt
		}
	}`, refreshToken)
		require.Equal(a, http.StatusOK, firstRec.Code, firstRec.Body.String())

		var firstResp struct {
			Data struct {
				RegisterMobileDevice struct {
					ID              string  `json:"id"`
					PushTokenPrefix string  `json:"pushTokenPrefix"`
					Platform        string  `json:"platform"`
					DeviceLabel     string  `json:"deviceLabel"`
					RevokedAt       *string `json:"revokedAt"`
				} `json:"registerMobileDevice"`
			} `json:"data"`
			Errors []any `json:"errors"`
		}
		require.NoError(a, json.Unmarshal(firstRec.Body.Bytes(), &firstResp))
		require.Empty(a, firstResp.Errors)
		require.NotEmpty(a, firstResp.Data.RegisterMobileDevice.ID)
		require.Equal(a, "Exponent", firstResp.Data.RegisterMobileDevice.PushTokenPrefix)
		require.Equal(a, "ios", firstResp.Data.RegisterMobileDevice.Platform)
		require.Equal(a, "iPhone", firstResp.Data.RegisterMobileDevice.DeviceLabel)
		require.Nil(a, firstResp.Data.RegisterMobileDevice.RevokedAt)

		secondRec := postGraphQLWithBearer(t, handler, `mutation {
		registerMobileDevice(input: {
			expoPushToken: "`+token+`"
			platform: "ios"
			deviceLabel: "iPhone updated"
		}) {
			id
			deviceLabel
			revokedAt
		}
	}`, refreshToken)
		require.Equal(a, http.StatusOK, secondRec.Code, secondRec.Body.String())

		var secondResp struct {
			Data struct {
				RegisterMobileDevice struct {
					ID          string  `json:"id"`
					DeviceLabel string  `json:"deviceLabel"`
					RevokedAt   *string `json:"revokedAt"`
				} `json:"registerMobileDevice"`
			} `json:"data"`
			Errors []any `json:"errors"`
		}
		require.NoError(a, json.Unmarshal(secondRec.Body.Bytes(), &secondResp))
		require.Empty(a, secondResp.Errors)
		require.Equal(a, firstResp.Data.RegisterMobileDevice.ID, secondResp.Data.RegisterMobileDevice.ID)
		require.Equal(a, "iPhone updated", secondResp.Data.RegisterMobileDevice.DeviceLabel)
		require.Nil(a, secondResp.Data.RegisterMobileDevice.RevokedAt)
	})
}

func TestGraphQLMobileDevicesListAndRevoke(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, _, cleanup := newTestHandler(t)
		defer cleanup()

		sessionCookie := bootstrapAdmin(t, handler)
		refreshToken := bootstrapMobileRefreshToken(t, handler, sessionCookie)

		registerRec := postGraphQLWithBearer(t, handler, `mutation {
		registerMobileDevice(input: {
			expoPushToken: "ExponentPushToken[list-revoke]"
			platform: "android"
			deviceLabel: "Pixel"
		}) {
			id
		}
	}`, refreshToken)
		require.Equal(a, http.StatusOK, registerRec.Code, registerRec.Body.String())

		var registerResp struct {
			Data struct {
				RegisterMobileDevice struct {
					ID string `json:"id"`
				} `json:"registerMobileDevice"`
			} `json:"data"`
			Errors []any `json:"errors"`
		}
		require.NoError(a, json.Unmarshal(registerRec.Body.Bytes(), &registerResp))
		require.Empty(a, registerResp.Errors)
		deviceID := registerResp.Data.RegisterMobileDevice.ID

		listRec := postGraphQL(t, handler, `{
		mobileDevices {
			id
			pushTokenPrefix
			platform
			deviceLabel
			revokedAt
		}
	}`, sessionCookie)
		require.Equal(a, http.StatusOK, listRec.Code, listRec.Body.String())

		var listResp struct {
			Data struct {
				MobileDevices []struct {
					ID              string  `json:"id"`
					PushTokenPrefix string  `json:"pushTokenPrefix"`
					Platform        string  `json:"platform"`
					DeviceLabel     string  `json:"deviceLabel"`
					RevokedAt       *string `json:"revokedAt"`
				} `json:"mobileDevices"`
			} `json:"data"`
			Errors []any `json:"errors"`
		}
		require.NoError(a, json.Unmarshal(listRec.Body.Bytes(), &listResp))
		require.Empty(a, listResp.Errors)
		require.Len(a, listResp.Data.MobileDevices, 1)
		require.Equal(a, deviceID, listResp.Data.MobileDevices[0].ID)
		require.Equal(a, "android", listResp.Data.MobileDevices[0].Platform)
		require.Nil(a, listResp.Data.MobileDevices[0].RevokedAt)

		revokeRec := postGraphQL(t, handler, `mutation {
		revokeMobileDevice(id: "`+deviceID+`") {
			id
			revokedAt
		}
	}`, sessionCookie)
		require.Equal(a, http.StatusOK, revokeRec.Code, revokeRec.Body.String())

		var revokeResp struct {
			Data struct {
				RevokeMobileDevice struct {
					ID        string `json:"id"`
					RevokedAt string `json:"revokedAt"`
				} `json:"revokeMobileDevice"`
			} `json:"data"`
			Errors []any `json:"errors"`
		}
		require.NoError(a, json.Unmarshal(revokeRec.Body.Bytes(), &revokeResp))
		require.Empty(a, revokeResp.Errors)
		require.Equal(a, deviceID, revokeResp.Data.RevokeMobileDevice.ID)
		require.NotEmpty(a, revokeResp.Data.RevokeMobileDevice.RevokedAt)
	})
}

func bootstrapMobileRefreshToken(t *testing.T, handler http.Handler, sessionCookie *http.Cookie) string {
	t.Helper()

	codeRec := postMobileAuthCode(t, handler, sessionCookie)
	require.Equal(t, http.StatusOK, codeRec.Code)

	var codeResp struct {
		Code string `json:"code"`
	}
	require.NoError(t, json.Unmarshal(codeRec.Body.Bytes(), &codeResp))

	exchangeRec := postMobileAuthExchange(t, handler, codeResp.Code)
	require.Equal(t, http.StatusOK, exchangeRec.Code)

	var exchangeResp struct {
		RefreshToken string `json:"refresh_token"`
	}
	require.NoError(t, json.Unmarshal(exchangeRec.Body.Bytes(), &exchangeResp))
	require.NotEmpty(t, exchangeResp.RefreshToken)
	return exchangeResp.RefreshToken
}
