package queue

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkerLoggingMiddlewareLogsStartAndFinish(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	middleware := NewWorkerLoggingMiddleware(logger)
	job := &rivertype.JobRow{
		ID:      42,
		Kind:    HeartbeatScanArgs{}.Kind(),
		Queue:   river.QueueDefault,
		Attempt: 1,
	}

	err := middleware.Work(context.Background(), job, func(context.Context) error {
		return nil
	})
	require.NoError(t, err)

	lines := bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n"))
	require.Len(t, lines, 2)

	var started map[string]any
	require.NoError(t, json.Unmarshal(lines[0], &started))
	assert.Equal(t, "job started", started["msg"])
	assert.Equal(t, float64(42), started["job_id"])
	assert.Equal(t, "heartbeat_scan", started["job_kind"])
	assert.Equal(t, river.QueueDefault, started["queue"])
	assert.Equal(t, float64(1), started["attempt"])

	var finished map[string]any
	require.NoError(t, json.Unmarshal(lines[1], &finished))
	assert.Equal(t, "job finished", finished["msg"])
	assert.Equal(t, float64(42), started["job_id"])
	assert.Contains(t, finished, "duration_ms")
}

func TestWorkerLoggingMiddlewareLogsErrors(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	middleware := NewWorkerLoggingMiddleware(logger)
	job := &rivertype.JobRow{
		ID:      7,
		Kind:    HeartbeatScanArgs{}.Kind(),
		Queue:   river.QueueDefault,
		Attempt: 2,
	}

	workErr := errors.New("boom")
	err := middleware.Work(context.Background(), job, func(context.Context) error {
		return workErr
	})
	require.ErrorIs(t, err, workErr)

	lines := bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n"))
	require.Len(t, lines, 2)

	var finished map[string]any
	require.NoError(t, json.Unmarshal(lines[1], &finished))
	assert.Equal(t, "job finished", finished["msg"])
	assert.Equal(t, "boom", finished["error"])
}
