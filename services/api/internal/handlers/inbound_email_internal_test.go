package handlers

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInboundEmailAddress(t *testing.T) {
	require.Equal(t, "token@inbound.example.com", InboundEmailAddress("token", "inbound.example.com"))
	require.Empty(t, InboundEmailAddress("", "inbound.example.com"))
}

func TestIntegrationKeyTokenFromRecipient(t *testing.T) {
	token, err := integrationKeyTokenFromRecipient("token@inbound.example.com", "inbound.example.com")
	require.NoError(t, err)
	require.Equal(t, "token", token)

	token, err = integrationKeyTokenFromRecipient("Escalite <token@inbound.example.com>", "inbound.example.com")
	require.NoError(t, err)
	require.Equal(t, "token", token)

	_, err = integrationKeyTokenFromRecipient("token@other.example.com", "inbound.example.com")
	require.Error(t, err)
}
