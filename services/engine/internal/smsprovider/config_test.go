package smsprovider_test

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mdg-labs/escalite/services/engine/internal/crypto"
	"github.com/mdg-labs/escalite/services/engine/internal/smsprovider"
)

const testEncryptionKey = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func TestLoadConfigNoProvider(t *testing.T) {
	t.Setenv("ESCALITE_SMS_PROVIDER", "")

	cfg, err := smsprovider.LoadConfig()
	require.NoError(t, err)
	assert.Empty(t, cfg.ProviderName)
	assert.Nil(t, cfg.Twilio)
}

func TestLoadConfigTwilioProvider(t *testing.T) {
	t.Setenv("ESCALITE_SMS_PROVIDER", "twilio")
	t.Setenv("ESCALITE_TWILIO_ACCOUNT_SID", "AC123")
	t.Setenv("ESCALITE_TWILIO_AUTH_TOKEN", "secret")
	t.Setenv("ESCALITE_TWILIO_FROM_NUMBER", "+15551234567")

	cfg, err := smsprovider.LoadConfig()
	require.NoError(t, err)
	assert.Equal(t, smsprovider.ProviderTwilio, cfg.ProviderName)
	require.NotNil(t, cfg.Twilio)
	assert.Equal(t, "AC123", cfg.Twilio.AccountSID)
	assert.Equal(t, "secret", cfg.Twilio.AuthToken)
	assert.Equal(t, "+15551234567", cfg.Twilio.FromNumber)
	assert.Equal(t, "+15551234567", cfg.Twilio.VoiceFromNumber)
}

func TestLoadConfigTwilioVoiceFromOverride(t *testing.T) {
	t.Setenv("ESCALITE_SMS_PROVIDER", "twilio")
	t.Setenv("ESCALITE_TWILIO_ACCOUNT_SID", "AC123")
	t.Setenv("ESCALITE_TWILIO_AUTH_TOKEN", "secret")
	t.Setenv("ESCALITE_TWILIO_FROM_NUMBER", "+15551234567")
	t.Setenv("ESCALITE_TWILIO_VOICE_FROM_NUMBER", "+15557654321")

	cfg, err := smsprovider.LoadConfig()
	require.NoError(t, err)
	require.NotNil(t, cfg.Twilio)
	assert.Equal(t, "+15557654321", cfg.Twilio.VoiceFromNumber)
}

func TestLoadConfigRejectsUnknownProvider(t *testing.T) {
	t.Setenv("ESCALITE_SMS_PROVIDER", "pagerduty")

	_, err := smsprovider.LoadConfig()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ESCALITE_SMS_PROVIDER")
	assert.Contains(t, err.Error(), "twilio")
}

func TestTwilioConfigured(t *testing.T) {
	assert.False(t, smsprovider.TwilioConfigured(nil))
	assert.False(t, smsprovider.TwilioConfigured(&smsprovider.TwilioConfig{
		AccountSID: "AC123",
		AuthToken:  "secret",
	}))
	assert.True(t, smsprovider.TwilioConfigured(&smsprovider.TwilioConfig{
		AccountSID: "AC123",
		AuthToken:  "secret",
		FromNumber: "+15551234567",
	}))
}

func TestResolveTwilioAuthTokenFromEnv(t *testing.T) {
	token, err := smsprovider.ResolveTwilioAuthToken(nil, &smsprovider.TwilioConfig{
		AuthToken: "secret",
	}, nil)
	require.NoError(t, err)
	assert.Equal(t, "secret", token)
}

func TestResolveTwilioAuthTokenFromEncryptedStorage(t *testing.T) {
	key, err := hex.DecodeString(testEncryptionKey)
	require.NoError(t, err)
	box, err := crypto.NewBox(key)
	require.NoError(t, err)

	encrypted, err := box.Encrypt([]byte("encrypted-secret"))
	require.NoError(t, err)

	token, err := smsprovider.ResolveTwilioAuthToken(box, nil, &encrypted)
	require.NoError(t, err)
	assert.Equal(t, "encrypted-secret", token)
}
