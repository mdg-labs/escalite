package statuspageapi_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/mdg-labs/escalite/services/engine/statuspageapi"
)

var testSigningKey = []byte("0123456789abcdef0123456789abcdef")

func TestSignAndParseUnsubscribeTokenRoundTrip(t *testing.T) {
	subscriptionID := uuid.Must(uuid.NewV7())
	organizationID := uuid.Must(uuid.NewV7())

	token, err := statuspageapi.SignUnsubscribeToken(testSigningKey, statuspageapi.UnsubscribeClaims{
		SubscriptionID: subscriptionID,
		OrganizationID: organizationID,
	})
	require.NoError(t, err)

	claims, err := statuspageapi.ParseUnsubscribeToken(testSigningKey, token)
	require.NoError(t, err)
	require.Equal(t, subscriptionID, claims.SubscriptionID)
	require.Equal(t, organizationID, claims.OrganizationID)
}

func TestParseUnsubscribeTokenRejectsTamperedSignature(t *testing.T) {
	subscriptionID := uuid.Must(uuid.NewV7())
	organizationID := uuid.Must(uuid.NewV7())

	token, err := statuspageapi.SignUnsubscribeToken(testSigningKey, statuspageapi.UnsubscribeClaims{
		SubscriptionID: subscriptionID,
		OrganizationID: organizationID,
	})
	require.NoError(t, err)

	tampered := token[:len(token)-1] + "x"
	_, err = statuspageapi.ParseUnsubscribeToken(testSigningKey, tampered)
	require.ErrorIs(t, err, statuspageapi.ErrInvalidUnsubscribeToken)
}

func TestBuildUnsubscribeURLUsesDefaultBaseWhenUnset(t *testing.T) {
	require.Equal(
		t,
		"http://localhost:5174/unsubscribe?token=abc",
		statuspageapi.BuildUnsubscribeURL("", "abc"),
	)
}

func TestBuildUnsubscribeURLEscapesToken(t *testing.T) {
	require.Equal(
		t,
		"https://status.example.com/unsubscribe?token=abc%2Bdef",
		statuspageapi.BuildUnsubscribeURL("https://status.example.com/", "abc+def"),
	)
}
