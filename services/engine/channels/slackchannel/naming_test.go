package slackchannel_test

import (
	"testing"

	allure "github.com/allure-framework/allure-go/commons/gotest"

	"github.com/google/uuid"
	"github.com/allure-framework/allure-go/testify/require"

	"github.com/mdg-labs/escalite/services/engine/channels/slackchannel"
)

func TestRenderChannelNameUsesShortID(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		incidentID := uuid.MustParse("01934f5a-7b2c-7000-8000-000000000001")

		name := slackchannel.RenderChannelName("incident-{short_id}", incidentID, "Checkout degradation")
		require.Equal(a, "incident-01934f5a", name)
	})
}

func TestRenderChannelNameDefaultsTemplate(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		incidentID := uuid.MustParse("01934f5a-7b2c-7000-8000-000000000001")

		name := slackchannel.RenderChannelName("", incidentID, "Checkout degradation")
		require.Equal(a, "incident-01934f5a", name)
	})
}

func TestSanitizeChannelName(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		require.Equal(a, "incident-abc123", slackchannel.SanitizeChannelName("#Incident-ABC123!"))
		require.Equal(a, "incident", slackchannel.SanitizeChannelName("###"))
	})
}
