package smsprovider

import (
	"fmt"
	"os"
	"strings"

	"github.com/mdg-labs/escalite/services/engine/internal/crypto"
)

const (
	ProviderTwilio = "twilio"

	envSMSProvider          = "ESCALITE_SMS_PROVIDER"
	envTwilioAccountSID     = "ESCALITE_TWILIO_ACCOUNT_SID"
	envTwilioAuthToken      = "ESCALITE_TWILIO_AUTH_TOKEN"
	envTwilioFromNumber     = "ESCALITE_TWILIO_FROM_NUMBER"
	envTwilioVoiceFromNumber = "ESCALITE_TWILIO_VOICE_FROM_NUMBER"
)

// TwilioConfig holds Twilio account credentials for outbound SMS and voice.
type TwilioConfig struct {
	AccountSID     string
	AuthToken      string
	FromNumber     string
	VoiceFromNumber string
}

// Config holds parsed SMS/voice provider settings from the environment.
type Config struct {
	ProviderName string
	Twilio       *TwilioConfig
}

// LoadConfig reads SMS/voice provider settings from the environment.
// When ESCALITE_SMS_PROVIDER is unset, SMS/voice channels stay disabled.
func LoadConfig() (Config, error) {
	providerName := strings.ToLower(strings.TrimSpace(envOrDefault(envSMSProvider, "")))
	if providerName == "" {
		return Config{}, nil
	}

	switch providerName {
	case ProviderTwilio:
		twilioCfg, err := loadTwilioConfig()
		if err != nil {
			return Config{}, err
		}
		return Config{
			ProviderName: ProviderTwilio,
			Twilio:       twilioCfg,
		}, nil
	default:
		return Config{}, fmt.Errorf(
			"invalid %s %q: supported values are %q (see .env.example)",
			envSMSProvider,
			providerName,
			ProviderTwilio,
		)
	}
}

// TwilioConfigured reports whether Twilio credentials are complete.
func TwilioConfigured(cfg *TwilioConfig) bool {
	if cfg == nil {
		return false
	}
	return strings.TrimSpace(cfg.AccountSID) != "" &&
		strings.TrimSpace(cfg.AuthToken) != "" &&
		strings.TrimSpace(cfg.FromNumber) != ""
}

// ResolveTwilioAuthToken returns the plaintext auth token from env config or encrypted storage.
func ResolveTwilioAuthToken(box *crypto.Box, cfg *TwilioConfig, encrypted *crypto.Encrypted) (string, error) {
	if encrypted != nil && len(encrypted.Ciphertext) > 0 {
		if box == nil {
			return "", fmt.Errorf("encryption is not configured")
		}
		plaintext, err := box.Decrypt(*encrypted)
		if err != nil {
			return "", fmt.Errorf("decrypt twilio auth token: %w", err)
		}
		return string(plaintext), nil
	}
	if cfg == nil {
		return "", fmt.Errorf("twilio auth token is not configured")
	}
	token := strings.TrimSpace(cfg.AuthToken)
	if token == "" {
		return "", fmt.Errorf("twilio auth token is not configured")
	}
	return token, nil
}

func loadTwilioConfig() (*TwilioConfig, error) {
	accountSID := strings.TrimSpace(envOrDefault(envTwilioAccountSID, ""))
	authToken := strings.TrimSpace(envOrDefault(envTwilioAuthToken, ""))
	fromNumber := strings.TrimSpace(envOrDefault(envTwilioFromNumber, ""))
	voiceFrom := strings.TrimSpace(envOrDefault(envTwilioVoiceFromNumber, ""))
	if voiceFrom == "" {
		voiceFrom = fromNumber
	}

	if accountSID == "" || authToken == "" || fromNumber == "" {
		return &TwilioConfig{
			AccountSID:      accountSID,
			AuthToken:       authToken,
			FromNumber:      fromNumber,
			VoiceFromNumber: voiceFrom,
		}, nil
	}

	return &TwilioConfig{
		AccountSID:      accountSID,
		AuthToken:       authToken,
		FromNumber:      fromNumber,
		VoiceFromNumber: voiceFrom,
	}, nil
}

func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
