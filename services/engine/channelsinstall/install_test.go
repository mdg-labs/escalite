package channelsinstall_test

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mdg-labs/escalite/services/engine/channels"
	"github.com/mdg-labs/escalite/services/engine/channelsinstall"
	"github.com/mdg-labs/escalite/services/engine/internal/smsprovider"
)

func TestConfigureSMS(t *testing.T) {
	t.Run("leaves channels disabled without credentials", func(t *testing.T) {
		channelsinstall.ConfigureSMS(slog.Default(), smsprovider.Config{
			ProviderName: smsprovider.ProviderTwilio,
			Twilio: &smsprovider.TwilioConfig{
				AccountSID: "AC123",
			},
		})

		_, err := channels.Get("sms")
		require.Error(t, err)
		_, err = channels.Get("voice")
		require.Error(t, err)
	})

	t.Run("registers channels when twilio configured", func(t *testing.T) {
		channelsinstall.ConfigureSMS(slog.Default(), smsprovider.Config{
			ProviderName: smsprovider.ProviderTwilio,
			Twilio: &smsprovider.TwilioConfig{
				AccountSID: "AC123",
				AuthToken:  "secret",
				FromNumber: "+15551234567",
			},
		})

		_, err := channels.Get("sms")
		require.NoError(t, err)
		_, err = channels.Get("voice")
		require.NoError(t, err)
	})
}
