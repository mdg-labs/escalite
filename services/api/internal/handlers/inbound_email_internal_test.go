package handlers

import (
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"testing"

	"github.com/allure-framework/allure-go/testify/require"
)

func TestInboundEmailAddress(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		require.Equal(a, "token@inbound.example.com", InboundEmailAddress("token", "inbound.example.com"))
		require.Empty(a, InboundEmailAddress("", "inbound.example.com"))
	})
}

func TestIntegrationKeyTokenFromRecipient(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		token, err := integrationKeyTokenFromRecipient("token@inbound.example.com", "inbound.example.com")
		require.NoError(a, err)
		require.Equal(a, "token", token)

		token, err = integrationKeyTokenFromRecipient("Escalite <token@inbound.example.com>", "inbound.example.com")
		require.NoError(a, err)
		require.Equal(a, "token", token)

		_, err = integrationKeyTokenFromRecipient("token@other.example.com", "inbound.example.com")
		require.Error(a, err)
	})
}
