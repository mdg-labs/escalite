// Package channelsinstall registers built-in notification channel plugins.
package channelsinstall

import (
	"github.com/mdg-labs/escalite/services/engine/channels"
	"github.com/mdg-labs/escalite/services/engine/channels/email"
	"github.com/mdg-labs/escalite/services/engine/channels/push"
	"github.com/mdg-labs/escalite/services/engine/channels/slackdm"
	"github.com/mdg-labs/escalite/services/engine/channels/webhook"
	engineemail "github.com/mdg-labs/escalite/services/engine/internal/email"
)

// ConfigureEmail sets the SMTP sender used by the email notification channel.
func ConfigureEmail(sender engineemail.Sender) {
	email.SetSender(sender)
}

func init() {
	channels.Register(email.New())
	channels.Register(push.New())
	channels.Register(webhook.New())
	channels.Register(slackdm.New())
}
