package sms_test

import (
	"encoding/json"
	"testing"

	allure "github.com/allure-framework/allure-go/commons/gotest"

	"github.com/allure-framework/allure-go/testify/require"

	"github.com/mdg-labs/escalite/services/engine/channels"
	"github.com/mdg-labs/escalite/services/engine/channels/sms"
)

func TestValidateConfigRequiresE164PhoneNumber(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		channel := sms.New()

		err := channel.ValidateConfig(json.RawMessage(`{"phone_number":"5551234567"}`))
		require.Error(a, err)
		require.Contains(a, err.Error(), "E.164")

		err = channel.ValidateConfig(json.RawMessage(`{"phone_number":"+15551234567"}`))
		require.NoError(a, err)
	})
}

func TestConfigSchemaReturnsJSON(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		channel := sms.New()
		schema := channel.ConfigSchema()
		require.Contains(a, string(schema), `"phone_number"`)
	})
}

func TestName(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		channel := sms.New()
		require.Equal(a, "sms", channel.Name())
	})
}

func TestValidateConfigUsesSharedValidator(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		err := channels.ValidateConfig("sms", json.RawMessage(`{"phone_number":"+15551234567"}`))
		require.Error(a, err)
	})
}
