package queue

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"

	allure "github.com/allure-framework/allure-go/commons/gotest"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	"github.com/allure-framework/allure-go/testify/assert"
	"github.com/allure-framework/allure-go/testify/require"
)

func TestWorkerLoggingMiddlewareLogsStartAndFinish(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

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
		require.NoError(a, err)

		lines := bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n"))
		require.Len(a, lines, 2)

		var started map[string]any
		require.NoError(a, json.Unmarshal(lines[0], &started))
		assert.Equal(a, "job started", started["msg"])
		assert.Equal(a, float64(42), started["job_id"])
		assert.Equal(a, "heartbeat_scan", started["job_kind"])
		assert.Equal(a, river.QueueDefault, started["queue"])
		assert.Equal(a, float64(1), started["attempt"])

		var finished map[string]any
		require.NoError(a, json.Unmarshal(lines[1], &finished))
		assert.Equal(a, "job finished", finished["msg"])
		assert.Equal(a, float64(42), started["job_id"])
		assert.Contains(a, finished, "duration_ms")
	})
}

func TestWorkerLoggingMiddlewareLogsErrors(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

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
		require.ErrorIs(a, err, workErr)

		lines := bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n"))
		require.Len(a, lines, 2)

		var finished map[string]any
		require.NoError(a, json.Unmarshal(lines[1], &finished))
		assert.Equal(a, "job finished", finished["msg"])
		assert.Equal(a, "boom", finished["error"])
	})
}
