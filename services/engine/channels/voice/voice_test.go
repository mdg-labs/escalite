package voice_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mdg-labs/escalite/services/engine/channels"
	"github.com/mdg-labs/escalite/services/engine/channels/voice"
	"github.com/mdg-labs/escalite/services/engine/internal/smsprovider"
)

type stubProvider struct {
	lastVoice smsprovider.VoiceParams
}

func (s *stubProvider) Name() string { return "stub" }

func (s *stubProvider) SendSMS(context.Context, smsprovider.SMSParams) error {
	return nil
}

func (s *stubProvider) MakeVoiceCall(_ context.Context, params smsprovider.VoiceParams) error {
	s.lastVoice = params
	return nil
}

func TestSendBuildsVoiceMessageWithAlertTitleAndService(t *testing.T) {
	provider := &stubProvider{}
	voice.SetProvider(provider)
	t.Cleanup(func() { voice.SetProvider(nil) })

	channel := voice.New()
	config, err := json.Marshal(map[string]string{
		"phone_number": "+15551234567",
	})
	require.NoError(t, err)

	err = channel.Send(context.Background(), channels.SendParams{
		Alert: channels.Alert{
			Summary:     "Disk full",
			ServiceName: "checkout-api",
		},
		Config: config,
	})
	require.NoError(t, err)
	require.Equal(t, smsprovider.VoiceParams{
		To:      "+15551234567",
		Message: "Disk full on checkout-api",
	}, provider.lastVoice)
}
