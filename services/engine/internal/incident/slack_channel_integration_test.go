package incident_test

import (
	"context"
	"encoding/hex"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/mdg-labs/escalite/services/engine/channels/slackchannel"
	"github.com/mdg-labs/escalite/services/engine/internal/crypto"
	"github.com/mdg-labs/escalite/services/engine/internal/db"
	"github.com/mdg-labs/escalite/services/engine/internal/escalation"
	"github.com/mdg-labs/escalite/services/engine/internal/incident"
	"github.com/mdg-labs/escalite/services/engine/internal/queue"
	"github.com/mdg-labs/escalite/services/engine/internal/testutil"
	_ "github.com/mdg-labs/escalite/services/engine/channelsinstall"
)

const testEncryptionKeyHex = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

type slackRoundTripFunc func(*http.Request) (*http.Response, error)

func (fn slackRoundTripFunc) Do(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func testSecretsBox(t *testing.T) *crypto.Box {
	t.Helper()
	key, err := hex.DecodeString(testEncryptionKeyHex)
	require.NoError(t, err)
	box, err := crypto.NewBox(key)
	require.NoError(t, err)
	return box
}

func TestAutoPromoteCreatesSlackChannel(t *testing.T) {
	slackchannel.SetHTTPClient(slackRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		require.Equal(t, "https://slack.com/api/conversations.create", req.URL.String())
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"ok":true,"channel":{"id":"CAUTOPROMO"}}`)),
			Header:     make(http.Header),
		}, nil
	}))
	t.Cleanup(func() { slackchannel.SetHTTPClient(nil) })

	ctx := context.Background()
	databaseURL, cleanup, err := testutil.StartPostgres(ctx)
	require.NoError(t, err)
	defer cleanup()

	logger := slog.Default()
	require.NoError(t, testutil.MigrateUp(ctx, databaseURL, logger))

	secrets := testSecretsBox(t)
	incident.ConfigureSlackChannels(incident.SlackChannelConfig{
		Template: "incident-{short_id}",
		Secrets:  secrets,
		Logger:   logger,
	})
	t.Cleanup(func() { incident.ConfigureSlackChannels(incident.SlackChannelConfig{}) })

	queueClient, err := queue.New(ctx, queue.Options{
		DatabaseURL:   databaseURL,
		Logger:        logger,
		EncryptionKey: mustDecodeHexKey(t),
	})
	require.NoError(t, err)
	defer queueClient.Close()

	require.NoError(t, queueClient.Start(ctx))
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = queueClient.Stop(stopCtx)
	}()

	queries := db.New(queueClient.Pool())
	orgID, serviceID, _ := seedAutoPromoteService(t, ctx, queueClient.Pool(), queries, 1)

	encrypted, err := secrets.Encrypt([]byte("xoxb-autopromote-token"))
	require.NoError(t, err)
	_, err = queueClient.Pool().Exec(ctx, `
		INSERT INTO organization_slack_settings (
			organization_id,
			bot_token_ciphertext,
			encryption_key_id,
			token_hint
		) VALUES ($1, $2, $3, $4)
	`, orgID, encrypted.Ciphertext, encrypted.KeyID, "oken")
	require.NoError(t, err)

	_, err = escalation.CreateTriggeredAlert(ctx, queueClient.Pool(), queueClient, escalation.CreateTriggeredAlertParams{
		AlertID:        uuid.Must(uuid.NewV7()),
		OrganizationID: orgID,
		ServiceID:      serviceID,
		DedupKey:       "auto-promote-slack",
		Summary:        "Checkout degradation",
		Priority:       "high",
	})
	require.NoError(t, err)

	var slackChannelID *string
	err = queueClient.Pool().QueryRow(ctx, `
		SELECT slack_channel_id
		FROM incidents
		WHERE organization_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`, orgID).Scan(&slackChannelID)
	require.NoError(t, err)
	require.NotNil(t, slackChannelID)
	require.Equal(t, "CAUTOPROMO", *slackChannelID)
}

func TestAutoPromoteKeepsIncidentWhenSlackChannelFails(t *testing.T) {
	slackchannel.SetHTTPClient(slackRoundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"ok":false,"error":"invalid_auth"}`)),
			Header:     make(http.Header),
		}, nil
	}))
	t.Cleanup(func() { slackchannel.SetHTTPClient(nil) })

	ctx := context.Background()
	databaseURL, cleanup, err := testutil.StartPostgres(ctx)
	require.NoError(t, err)
	defer cleanup()

	logger := slog.Default()
	require.NoError(t, testutil.MigrateUp(ctx, databaseURL, logger))

	secrets := testSecretsBox(t)
	incident.ConfigureSlackChannels(incident.SlackChannelConfig{
		Template: "incident-{short_id}",
		Secrets:  secrets,
		Logger:   logger,
	})
	t.Cleanup(func() { incident.ConfigureSlackChannels(incident.SlackChannelConfig{}) })

	queueClient, err := queue.New(ctx, queue.Options{
		DatabaseURL:   databaseURL,
		Logger:        logger,
		EncryptionKey: mustDecodeHexKey(t),
	})
	require.NoError(t, err)
	defer queueClient.Close()

	require.NoError(t, queueClient.Start(ctx))
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = queueClient.Stop(stopCtx)
	}()

	queries := db.New(queueClient.Pool())
	orgID, serviceID, _ := seedAutoPromoteService(t, ctx, queueClient.Pool(), queries, 1)

	encrypted, err := secrets.Encrypt([]byte("xoxb-autopromote-token"))
	require.NoError(t, err)
	_, err = queueClient.Pool().Exec(ctx, `
		INSERT INTO organization_slack_settings (
			organization_id,
			bot_token_ciphertext,
			encryption_key_id,
			token_hint
		) VALUES ($1, $2, $3, $4)
	`, orgID, encrypted.Ciphertext, encrypted.KeyID, "oken")
	require.NoError(t, err)

	_, err = escalation.CreateTriggeredAlert(ctx, queueClient.Pool(), queueClient, escalation.CreateTriggeredAlertParams{
		AlertID:        uuid.Must(uuid.NewV7()),
		OrganizationID: orgID,
		ServiceID:      serviceID,
		DedupKey:       "auto-promote-slack-fail",
		Summary:        "Checkout degradation",
		Priority:       "high",
	})
	require.NoError(t, err)

	var incidentCount int
	err = queueClient.Pool().QueryRow(ctx, `
		SELECT count(*)
		FROM incidents
		WHERE organization_id = $1
	`, orgID).Scan(&incidentCount)
	require.NoError(t, err)
	require.Equal(t, 1, incidentCount)

	var slackChannelID *string
	err = queueClient.Pool().QueryRow(ctx, `
		SELECT slack_channel_id
		FROM incidents
		WHERE organization_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`, orgID).Scan(&slackChannelID)
	require.NoError(t, err)
	require.Nil(t, slackChannelID)
}

func mustDecodeHexKey(t *testing.T) []byte {
	t.Helper()
	key, err := hex.DecodeString(testEncryptionKeyHex)
	require.NoError(t, err)
	return key
}
