package ratelimit_test

import (
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"testing"
	"time"

	"github.com/allure-framework/allure-go/testify/require"

	"github.com/mdg-labs/escalite/services/api/internal/ratelimit"
)

func TestMemoryLimiterBlocksAfterThreshold(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		limiter := ratelimit.NewMemoryLimiter(2, time.Minute)

		require.True(a, limiter.Allow("user@example.com"))
		require.True(a, limiter.Allow("user@example.com"))
		require.False(a, limiter.Allow("user@example.com"))
		require.True(a, limiter.Allow("other@example.com"))
	})
}
