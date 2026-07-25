package twilio

import (
	"context"
	"errors"
	"fmt"
	"strings"

	twiliogo "github.com/twilio/twilio-go"
	openapi "github.com/twilio/twilio-go/rest/api/v2010"

	"github.com/mdg-labs/escalite/services/engine/internal/smsprovider"
)

const providerName = smsprovider.ProviderTwilio

// Config holds Twilio credentials for outbound SMS and voice.
type Config struct {
	AccountSID      string
	AuthToken       string
	FromNumber      string
	VoiceFromNumber string
	// Client is optional; when nil a twilio-go client is built from credentials.
	Client *twiliogo.RestClient
}

// Provider is the Twilio-backed SMS/voice provider.
type Provider struct {
	cfg    Config
	client *twiliogo.RestClient
}

// New returns a Twilio provider configured with cfg.
func New(cfg Config) *Provider {
	voiceFrom := strings.TrimSpace(cfg.VoiceFromNumber)
	if voiceFrom == "" {
		voiceFrom = strings.TrimSpace(cfg.FromNumber)
	}
	normalized := Config{
		AccountSID:      strings.TrimSpace(cfg.AccountSID),
		AuthToken:       strings.TrimSpace(cfg.AuthToken),
		FromNumber:      strings.TrimSpace(cfg.FromNumber),
		VoiceFromNumber: voiceFrom,
		Client:          cfg.Client,
	}

	client := cfg.Client
	if client == nil {
		client = twiliogo.NewRestClientWithParams(twiliogo.ClientParams{
			Username:   normalized.AccountSID,
			Password:   normalized.AuthToken,
			AccountSid: normalized.AccountSID,
		})
	}

	return &Provider{cfg: normalized, client: client}
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
	message := strings.TrimSpace(params.Message)
	if message == "" {
		return errors.New("sms message is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	apiParams := &openapi.CreateMessageParams{}
	apiParams.SetTo(strings.TrimSpace(params.To))
	apiParams.SetFrom(p.cfg.FromNumber)
	apiParams.SetBody(message)

	if _, err := p.client.Api.CreateMessage(apiParams); err != nil {
		return fmt.Errorf("twilio send sms: %w", err)
	}
	return nil
}

func (p *Provider) MakeVoiceCall(ctx context.Context, params smsprovider.VoiceParams) error {
	if err := p.validateConfigured(); err != nil {
		return err
	}
	if err := validatePhoneNumber(params.To); err != nil {
		return fmt.Errorf("voice recipient: %w", err)
	}
	message := strings.TrimSpace(params.Message)
	if message == "" {
		return errors.New("voice message is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	apiParams := &openapi.CreateCallParams{}
	apiParams.SetTo(strings.TrimSpace(params.To))
	apiParams.SetFrom(p.cfg.VoiceFromNumber)
	apiParams.SetTwiml(buildTwiml(message))

	if _, err := p.client.Api.CreateCall(apiParams); err != nil {
		return fmt.Errorf("twilio voice call: %w", err)
	}
	return nil
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

func buildTwiml(message string) string {
	return fmt.Sprintf(
		`<?xml version="1.0" encoding="UTF-8"?><Response><Say>%s</Say></Response>`,
		escapeXML(message),
	)
}

func escapeXML(value string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&apos;",
	)
	return replacer.Replace(value)
}
