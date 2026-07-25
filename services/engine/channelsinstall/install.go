// Package channelsinstall registers built-in notification channel plugins.
package channelsinstall

import (
	"github.com/mdg-labs/escalite/services/engine/channels"
	"github.com/mdg-labs/escalite/services/engine/channels/email"
	"github.com/mdg-labs/escalite/services/engine/channels/push"
	"github.com/mdg-labs/escalite/services/engine/channels/slackdm"
	"github.com/mdg-labs/escalite/services/engine/channels/webhook"
)

func init() {
	channels.Register(email.New())
	channels.Register(push.New())
	channels.Register(webhook.New())
	channels.Register(slackdm.New())
}
