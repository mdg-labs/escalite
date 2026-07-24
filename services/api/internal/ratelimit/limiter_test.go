package ratelimit_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/mdg-labs/escalite/services/api/internal/ratelimit"
)

func TestMemoryLimiterBlocksAfterThreshold(t *testing.T) {
	limiter := ratelimit.NewMemoryLimiter(2, time.Minute)

	require.True(t, limiter.Allow("user@example.com"))
	require.True(t, limiter.Allow("user@example.com"))
	require.False(t, limiter.Allow("user@example.com"))
	require.True(t, limiter.Allow("other@example.com"))
}
