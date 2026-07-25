package queue

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHeartbeatScanArgsKind(t *testing.T) {
	assert.Equal(t, "heartbeat_scan", HeartbeatScanArgs{}.Kind())
}

func TestNewPeriodicJobsRegistersHeartbeatScan(t *testing.T) {
	jobs := NewPeriodicJobs(DefaultHeartbeatScanInterval)
	require.Len(t, jobs, 1)
	require.NotNil(t, jobs[0])
}
