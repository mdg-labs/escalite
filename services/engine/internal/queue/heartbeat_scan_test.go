package queue

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/riverqueue/river"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHeartbeatScanWorkerLogsCompletion(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	worker := NewHeartbeatScanWorker(logger)

	err := worker.Work(context.Background(), &river.Job[HeartbeatScanArgs]{})
	require.NoError(t, err)

	var entry map[string]any
	require.NoError(t, json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &entry))
	assert.Equal(t, "heartbeat scan completed", entry["msg"])
}

func TestHeartbeatScanArgsKind(t *testing.T) {
	assert.Equal(t, "heartbeat_scan", HeartbeatScanArgs{}.Kind())
}

func TestNewPeriodicJobsRegistersHeartbeatScan(t *testing.T) {
	jobs := NewPeriodicJobs(DefaultHeartbeatScanInterval)
	require.Len(t, jobs, 1)
	require.NotNil(t, jobs[0])
}
