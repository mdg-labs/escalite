package email_test

import (
	"context"
	"testing"

	allure "github.com/allure-framework/allure-go/commons/gotest"

	"github.com/allure-framework/allure-go/testify/require"

	"github.com/mdg-labs/escalite/services/engine/channels"
	emailchannel "github.com/mdg-labs/escalite/services/engine/channels/email"
	engineemail "github.com/mdg-labs/escalite/services/engine/internal/email"
)

type recordingSender struct {
	messages []engineemail.Message
	err      error
}

func (r *recordingSender) Send(_ context.Context, msg engineemail.Message) error {
	r.messages = append(r.messages, msg)
	return r.err
}

func TestSendBuildsMultipartAlertEmail(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		sender := &recordingSender{}
		emailchannel.SetSender(sender)
		a.T().Cleanup(func() { emailchannel.SetSender(nil) })

		channel := emailchannel.New()
		err := channel.Send(context.Background(), channels.SendParams{
			Target: channels.Target{
				Type:  "user",
				Email: "oncall@example.com",
			},
			Alert: channels.Alert{
				ID:          "alert-1",
				Summary:     "Disk full",
				Description: "Volume /data is 99% full",
				Priority:    "high",
				Status:      "triggered",
			},
		})
		require.NoError(a, err)
		require.Len(a, sender.messages, 1)
		require.Equal(a, "oncall@example.com", sender.messages[0].To)
		require.Equal(a, "[HIGH] Disk full", sender.messages[0].Subject)
		require.Contains(a, sender.messages[0].TextBody, "Disk full")
		require.Contains(a, sender.messages[0].HTMLBody, "Disk full")
		require.Contains(a, sender.messages[0].HTMLBody, "Volume /data is 99% full")
	})
}

func TestSendFailsWhenSMTPNotConfigured(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		emailchannel.SetSender(nil)
		channel := emailchannel.New()

		err := channel.Send(context.Background(), channels.SendParams{
			Target: channels.Target{Email: "oncall@example.com"},
			Alert:  channels.Alert{Summary: "Disk full", Priority: "high", Status: "triggered"},
		})
		require.Error(a, err)
		require.Contains(a, err.Error(), "smtp not configured")
	})
}

func TestSendFailsWithoutRecipientEmail(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		emailchannel.SetSender(&recordingSender{})
		a.T().Cleanup(func() { emailchannel.SetSender(nil) })

		channel := emailchannel.New()
		err := channel.Send(context.Background(), channels.SendParams{
			Target: channels.Target{Type: "user"},
			Alert:  channels.Alert{Summary: "Disk full", Priority: "high", Status: "triggered"},
		})
		require.Error(a, err)
		require.Contains(a, err.Error(), "recipient email is required")
	})
}
