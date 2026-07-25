package email_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

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
	sender := &recordingSender{}
	emailchannel.SetSender(sender)
	t.Cleanup(func() { emailchannel.SetSender(nil) })

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
	require.NoError(t, err)
	require.Len(t, sender.messages, 1)
	require.Equal(t, "oncall@example.com", sender.messages[0].To)
	require.Equal(t, "[HIGH] Disk full", sender.messages[0].Subject)
	require.Contains(t, sender.messages[0].TextBody, "Disk full")
	require.Contains(t, sender.messages[0].HTMLBody, "Disk full")
	require.Contains(t, sender.messages[0].HTMLBody, "Volume /data is 99% full")
}

func TestSendFailsWhenSMTPNotConfigured(t *testing.T) {
	emailchannel.SetSender(nil)
	channel := emailchannel.New()

	err := channel.Send(context.Background(), channels.SendParams{
		Target: channels.Target{Email: "oncall@example.com"},
		Alert:  channels.Alert{Summary: "Disk full", Priority: "high", Status: "triggered"},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "smtp not configured")
}

func TestSendFailsWithoutRecipientEmail(t *testing.T) {
	emailchannel.SetSender(&recordingSender{})
	t.Cleanup(func() { emailchannel.SetSender(nil) })

	channel := emailchannel.New()
	err := channel.Send(context.Background(), channels.SendParams{
		Target: channels.Target{Type: "user"},
		Alert:  channels.Alert{Summary: "Disk full", Priority: "high", Status: "triggered"},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "recipient email is required")
}
