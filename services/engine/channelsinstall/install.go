// Package channelsinstall registers built-in notification channel plugins.
package channelsinstall

import (
	"log/slog"

	"github.com/mdg-labs/escalite/services/engine/channels"
	"github.com/mdg-labs/escalite/services/engine/channels/email"
	"github.com/mdg-labs/escalite/services/engine/channels/push"
	"github.com/mdg-labs/escalite/services/engine/channels/slackdm"
	"github.com/mdg-labs/escalite/services/engine/channels/sms"
	"github.com/mdg-labs/escalite/services/engine/channels/voice"
	"github.com/mdg-labs/escalite/services/engine/channels/webhook"
	engineemail "github.com/mdg-labs/escalite/services/engine/internal/email"
	"github.com/mdg-labs/escalite/services/engine/internal/smsprovider"
	twilioprovider "github.com/mdg-labs/escalite/services/engine/internal/smsprovider/twilio"
)

// ConfigureEmail sets the SMTP sender used by the email notification channel.
func ConfigureEmail(sender engineemail.Sender) {
	email.SetSender(sender)
}

// ConfigureSMS wires the selected SMS/voice provider and registers sms/voice channels when configured.
func ConfigureSMS(logger *slog.Logger, cfg smsprovider.Config) {
	if logger == nil {
		logger = slog.Default()
	}

	switch cfg.ProviderName {
	case "":
		return
	case smsprovider.ProviderTwilio:
		if !smsprovider.TwilioConfigured(cfg.Twilio) {
			logger.Warn(
				"twilio sms/voice disabled: missing ESCALITE_TWILIO_ACCOUNT_SID, ESCALITE_TWILIO_AUTH_TOKEN, or ESCALITE_TWILIO_FROM_NUMBER (see .env.example)",
			)
			return
		}

		provider := twilioprovider.New(twilioprovider.Config{
			AccountSID:      cfg.Twilio.AccountSID,
			AuthToken:       cfg.Twilio.AuthToken,
			FromNumber:      cfg.Twilio.FromNumber,
			VoiceFromNumber: cfg.Twilio.VoiceFromNumber,
		})
		smsprovider.Register(provider)
		sms.SetProvider(provider)
		voice.SetProvider(provider)
		channels.Register(sms.New())
		channels.Register(voice.New())
		logger.Info("sms/voice provider configured", "provider", provider.Name())
	default:
		logger.Warn(
			"sms/voice notifications disabled: unsupported provider configured",
			"provider", cfg.ProviderName,
		)
	}
}

func init() {
	channels.Register(email.New())
	channels.Register(push.New())
	channels.Register(webhook.New())
	channels.Register(slackdm.New())
}
