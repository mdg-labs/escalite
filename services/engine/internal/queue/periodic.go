package queue

import (
	"time"

	"github.com/riverqueue/river"
)

const heartbeatScanPeriodicJobID = "heartbeat_scan"

// DefaultHeartbeatScanInterval is the noop scan cadence until Phase 1 makes it configurable.
const DefaultHeartbeatScanInterval = time.Minute

// NewPeriodicJobs registers engine periodic enqueue schedules.
func NewPeriodicJobs(scanInterval time.Duration) []*river.PeriodicJob {
	if scanInterval <= 0 {
		scanInterval = DefaultHeartbeatScanInterval
	}

	return []*river.PeriodicJob{
		river.NewPeriodicJob(
			river.PeriodicInterval(scanInterval),
			func() (river.JobArgs, *river.InsertOpts) {
				return HeartbeatScanArgs{}, nil
			},
			&river.PeriodicJobOpts{
				ID:         heartbeatScanPeriodicJobID,
				RunOnStart: true,
			},
		),
	}
}
