package sms_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mdg-labs/escalite/services/engine/channels"
	"github.com/mdg-labs/escalite/services/engine/channels/sms"
)

func TestValidateConfigRequiresE164PhoneNumber(t *testing.T) {
	channel := sms.New()

	err := channel.ValidateConfig(json.RawMessage(`{"phone_number":"5551234567"}`))
	require.Error(t, err)
	require.Contains(t, err.Error(), "E.164")

	err = channel.ValidateConfig(json.RawMessage(`{"phone_number":"+15551234567"}`))
	require.NoError(t, err)
}

func TestConfigSchemaReturnsJSON(t *testing.T) {
	channel := sms.New()
	schema := channel.ConfigSchema()
	require.Contains(t, string(schema), `"phone_number"`)
}

func TestName(t *testing.T) {
	channel := sms.New()
	require.Equal(t, "sms", channel.Name())
}

func TestValidateConfigUsesSharedValidator(t *testing.T) {
	err := channels.ValidateConfig("sms", json.RawMessage(`{"phone_number":"+15551234567"}`))
	require.Error(t, err)
}
