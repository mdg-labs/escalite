package smsprovider_test

import (
	"encoding/hex"
	"testing"

	allure "github.com/allure-framework/allure-go/commons/gotest"

	"github.com/allure-framework/allure-go/testify/assert"
	"github.com/allure-framework/allure-go/testify/require"

	"github.com/mdg-labs/escalite/services/engine/internal/crypto"
	"github.com/mdg-labs/escalite/services/engine/internal/smsprovider"
)

const testEncryptionKey = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func TestLoadConfigNoProvider(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		t.Setenv("ESCALITE_SMS_PROVIDER", "")

		cfg, err := smsprovider.LoadConfig()
		require.NoError(a, err)
		assert.Empty(a, cfg.ProviderName)
		assert.Nil(a, cfg.Twilio)
	})
}

func TestLoadConfigTwilioProvider(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		t.Setenv("ESCALITE_SMS_PROVIDER", "twilio")
		t.Setenv("ESCALITE_TWILIO_ACCOUNT_SID", "AC123")
		t.Setenv("ESCALITE_TWILIO_AUTH_TOKEN", "secret")
		t.Setenv("ESCALITE_TWILIO_FROM_NUMBER", "+15551234567")

		cfg, err := smsprovider.LoadConfig()
		require.NoError(a, err)
		assert.Equal(a, smsprovider.ProviderTwilio, cfg.ProviderName)
		require.NotNil(a, cfg.Twilio)
		assert.Equal(a, "AC123", cfg.Twilio.AccountSID)
		assert.Equal(a, "secret", cfg.Twilio.AuthToken)
		assert.Equal(a, "+15551234567", cfg.Twilio.FromNumber)
		assert.Equal(a, "+15551234567", cfg.Twilio.VoiceFromNumber)
	})
}

func TestLoadConfigTwilioVoiceFromOverride(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		t.Setenv("ESCALITE_SMS_PROVIDER", "twilio")
		t.Setenv("ESCALITE_TWILIO_ACCOUNT_SID", "AC123")
		t.Setenv("ESCALITE_TWILIO_AUTH_TOKEN", "secret")
		t.Setenv("ESCALITE_TWILIO_FROM_NUMBER", "+15551234567")
		t.Setenv("ESCALITE_TWILIO_VOICE_FROM_NUMBER", "+15557654321")

		cfg, err := smsprovider.LoadConfig()
		require.NoError(a, err)
		require.NotNil(a, cfg.Twilio)
		assert.Equal(a, "+15557654321", cfg.Twilio.VoiceFromNumber)
	})
}

func TestLoadConfigRejectsUnknownProvider(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		t.Setenv("ESCALITE_SMS_PROVIDER", "pagerduty")

		_, err := smsprovider.LoadConfig()
		require.Error(a, err)
		assert.Contains(a, err.Error(), "ESCALITE_SMS_PROVIDER")
		assert.Contains(a, err.Error(), "twilio")
	})
}

func TestTwilioConfigured(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		assert.False(a, smsprovider.TwilioConfigured(nil))
		assert.False(a, smsprovider.TwilioConfigured(&smsprovider.TwilioConfig{
			AccountSID: "AC123",
			AuthToken:  "secret",
		}))
		assert.True(a, smsprovider.TwilioConfigured(&smsprovider.TwilioConfig{
			AccountSID: "AC123",
			AuthToken:  "secret",
			FromNumber: "+15551234567",
		}))
	})
}

func TestResolveTwilioAuthTokenFromEnv(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		token, err := smsprovider.ResolveTwilioAuthToken(nil, &smsprovider.TwilioConfig{
			AuthToken: "secret",
		}, nil)
		require.NoError(a, err)
		assert.Equal(a, "secret", token)
	})
}

func TestResolveTwilioAuthTokenFromEncryptedStorage(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		key, err := hex.DecodeString(testEncryptionKey)
		require.NoError(a, err)
		box, err := crypto.NewBox(key)
		require.NoError(a, err)

		encrypted, err := box.Encrypt([]byte("encrypted-secret"))
		require.NoError(a, err)

		token, err := smsprovider.ResolveTwilioAuthToken(box, nil, &encrypted)
		require.NoError(a, err)
		assert.Equal(a, "encrypted-secret", token)
	})
}
