package slackchannel_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/mdg-labs/escalite/services/engine/channels/slackchannel"
)

func TestRenderChannelNameUsesShortID(t *testing.T) {
	incidentID := uuid.MustParse("01934f5a-7b2c-7000-8000-000000000001")

	name := slackchannel.RenderChannelName("incident-{short_id}", incidentID, "Checkout degradation")
	require.Equal(t, "incident-01934f5a", name)
}

func TestRenderChannelNameDefaultsTemplate(t *testing.T) {
	incidentID := uuid.MustParse("01934f5a-7b2c-7000-8000-000000000001")

	name := slackchannel.RenderChannelName("", incidentID, "Checkout degradation")
	require.Equal(t, "incident-01934f5a", name)
}

func TestSanitizeChannelName(t *testing.T) {
	require.Equal(t, "incident-abc123", slackchannel.SanitizeChannelName("#Incident-ABC123!"))
	require.Equal(t, "incident", slackchannel.SanitizeChannelName("###"))
}
