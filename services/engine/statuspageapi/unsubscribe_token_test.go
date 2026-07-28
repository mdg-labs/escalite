package statuspageapi_test

import (
	"testing"

	allure "github.com/allure-framework/allure-go/commons/gotest"

	"github.com/google/uuid"
	"github.com/allure-framework/allure-go/testify/require"

	"github.com/mdg-labs/escalite/services/engine/statuspageapi"
)

var testSigningKey = []byte("0123456789abcdef0123456789abcdef")

func TestSignAndParseUnsubscribeTokenRoundTrip(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		subscriptionID := uuid.Must(uuid.NewV7())
		organizationID := uuid.Must(uuid.NewV7())

		token, err := statuspageapi.SignUnsubscribeToken(testSigningKey, statuspageapi.UnsubscribeClaims{
			SubscriptionID: subscriptionID,
			OrganizationID: organizationID,
		})
		require.NoError(a, err)

		claims, err := statuspageapi.ParseUnsubscribeToken(testSigningKey, token)
		require.NoError(a, err)
		require.Equal(a, subscriptionID, claims.SubscriptionID)
		require.Equal(a, organizationID, claims.OrganizationID)
	})
}

func TestParseUnsubscribeTokenRejectsTamperedSignature(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		subscriptionID := uuid.Must(uuid.NewV7())
		organizationID := uuid.Must(uuid.NewV7())

		token, err := statuspageapi.SignUnsubscribeToken(testSigningKey, statuspageapi.UnsubscribeClaims{
			SubscriptionID: subscriptionID,
			OrganizationID: organizationID,
		})
		require.NoError(a, err)

		tampered := token[:len(token)-1] + "x"
		_, err = statuspageapi.ParseUnsubscribeToken(testSigningKey, tampered)
		require.ErrorIs(a, err, statuspageapi.ErrInvalidUnsubscribeToken)
	})
}

func TestBuildUnsubscribeURLUsesDefaultBaseWhenUnset(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		require.Equal(
			t,
			"http://localhost:5174/unsubscribe?token=abc",
			statuspageapi.BuildUnsubscribeURL("", "abc"),
		)
	})
}

func TestBuildUnsubscribeURLEscapesToken(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		require.Equal(
			t,
			"https://status.example.com/unsubscribe?token=abc%2Bdef",
			statuspageapi.BuildUnsubscribeURL("https://status.example.com/", "abc+def"),
		)
	})
}
