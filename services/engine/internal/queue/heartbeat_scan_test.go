package queue

import (
	"testing"

	allure "github.com/allure-framework/allure-go/commons/gotest"

	"github.com/allure-framework/allure-go/testify/assert"
	"github.com/allure-framework/allure-go/testify/require"
)

func TestHeartbeatScanArgsKind(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		assert.Equal(a, "heartbeat_scan", HeartbeatScanArgs{}.Kind())
	})
}

func TestNewPeriodicJobsRegistersHeartbeatScan(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		jobs := NewPeriodicJobs(DefaultHeartbeatScanInterval)
		require.Len(a, jobs, 1)
		require.NotNil(a, jobs[0])
	})
}
