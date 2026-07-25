package twilio_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mdg-labs/escalite/services/engine/internal/smsprovider"
	twilioprovider "github.com/mdg-labs/escalite/services/engine/internal/smsprovider/twilio"
)

func TestProviderName(t *testing.T) {
	provider := twilioprovider.New(twilioprovider.Config{
		AccountSID: "AC123",
		AuthToken:  "secret",
		FromNumber: "+15551234567",
	})
	assert.Equal(t, smsprovider.ProviderTwilio, provider.Name())
}

func TestSendSMSRequiresConfiguredProvider(t *testing.T) {
	provider := twilioprovider.New(twilioprovider.Config{})

	err := provider.SendSMS(context.Background(), smsprovider.SMSParams{
		To:      "+15551234567",
		Message: "hello",
	})
	require.ErrorIs(t, err, smsprovider.ErrNotConfigured)
}

func TestSendSMSValidatesRecipient(t *testing.T) {
	provider := twilioprovider.New(twilioprovider.Config{
		AccountSID: "AC123",
		AuthToken:  "secret",
		FromNumber: "+15551234567",
	})

	err := provider.SendSMS(context.Background(), smsprovider.SMSParams{
		To:      "5551234567",
		Message: "hello",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "E.164")
}

func TestSendSMSStubReturnsNotImplemented(t *testing.T) {
	provider := twilioprovider.New(twilioprovider.Config{
		AccountSID: "AC123",
		AuthToken:  "secret",
		FromNumber: "+15551234567",
	})

	err := provider.SendSMS(context.Background(), smsprovider.SMSParams{
		To:      "+15551234567",
		Message: "hello",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not implemented")
}
