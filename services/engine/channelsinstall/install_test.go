package channelsinstall_test

import (
	"log/slog"
	"testing"

	allure "github.com/allure-framework/allure-go/commons/gotest"

	"github.com/allure-framework/allure-go/testify/require"

	"github.com/mdg-labs/escalite/services/engine/channels"
	"github.com/mdg-labs/escalite/services/engine/channelsinstall"
	"github.com/mdg-labs/escalite/services/engine/internal/smsprovider"
)

func TestConfigureSMS(t *testing.T) {

	allure.Test(t, "leaves channels disabled without credentials", func(a *allure.Context) {

		channelsinstall.ConfigureSMS(slog.Default(), smsprovider.Config{
			ProviderName: smsprovider.ProviderTwilio,
			Twilio: &smsprovider.TwilioConfig{
				AccountSID: "AC123",
			},
		})

		_, err := channels.Get("sms")
		require.Error(a, err)
		_, err = channels.Get("voice")
		require.Error(a, err)
	
	})

	allure.Test(t, "registers channels when twilio configured", func(a *allure.Context) {

		channelsinstall.ConfigureSMS(slog.Default(), smsprovider.Config{
			ProviderName: smsprovider.ProviderTwilio,
			Twilio: &smsprovider.TwilioConfig{
				AccountSID: "AC123",
				AuthToken:  "secret",
				FromNumber: "+15551234567",
			},
		})

		_, err := channels.Get("sms")
		require.NoError(a, err)
		_, err = channels.Get("voice")
		require.NoError(a, err)
	
	})

}
