package twilio

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/mdg-labs/escalite/services/engine/internal/smsprovider"
)

const providerName = smsprovider.ProviderTwilio

// Config holds Twilio credentials for outbound SMS and voice.
type Config struct {
	AccountSID      string
	AuthToken       string
	FromNumber      string
	VoiceFromNumber string
}

// Provider is the Twilio-backed SMS/voice provider stub.
type Provider struct {
	cfg Config
}

// New returns a Twilio provider configured with cfg.
func New(cfg Config) *Provider {
	voiceFrom := strings.TrimSpace(cfg.VoiceFromNumber)
	if voiceFrom == "" {
		voiceFrom = strings.TrimSpace(cfg.FromNumber)
	}
	return &Provider{cfg: Config{
		AccountSID:      strings.TrimSpace(cfg.AccountSID),
		AuthToken:       strings.TrimSpace(cfg.AuthToken),
		FromNumber:      strings.TrimSpace(cfg.FromNumber),
		VoiceFromNumber: voiceFrom,
	}}
}

func (p *Provider) Name() string {
	return providerName
}

func (p *Provider) SendSMS(ctx context.Context, params smsprovider.SMSParams) error {
	if err := p.validateConfigured(); err != nil {
		return err
	}
	if err := validatePhoneNumber(params.To); err != nil {
		return fmt.Errorf("sms recipient: %w", err)
	}
	if strings.TrimSpace(params.Message) == "" {
		return errors.New("sms message is required")
	}
	_ = ctx
	return errors.New("twilio sms delivery is not implemented yet")
}

func (p *Provider) MakeVoiceCall(ctx context.Context, params smsprovider.VoiceParams) error {
	if err := p.validateConfigured(); err != nil {
		return err
	}
	if err := validatePhoneNumber(params.To); err != nil {
		return fmt.Errorf("voice recipient: %w", err)
	}
	if strings.TrimSpace(params.Message) == "" {
		return errors.New("voice message is required")
	}
	_ = ctx
	return errors.New("twilio voice delivery is not implemented yet")
}

func (p *Provider) validateConfigured() error {
	if !smsprovider.TwilioConfigured(&smsprovider.TwilioConfig{
		AccountSID: p.cfg.AccountSID,
		AuthToken:  p.cfg.AuthToken,
		FromNumber: p.cfg.FromNumber,
	}) {
		return smsprovider.ErrNotConfigured
	}
	return nil
}

func validatePhoneNumber(value string) error {
	phone := strings.TrimSpace(value)
	if phone == "" {
		return errors.New("phone number is required")
	}
	if !strings.HasPrefix(phone, "+") {
		return errors.New("phone number must be in E.164 format")
	}
	return nil
}
