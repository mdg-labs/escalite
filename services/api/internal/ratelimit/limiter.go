package ratelimit

import (
	"sync"
	"time"
)

// Limiter decides whether a keyed action is allowed within a sliding window.
type Limiter interface {
	Allow(key string) bool
}

// MemoryLimiter tracks request timestamps per key in memory.
type MemoryLimiter struct {
	mu     sync.Mutex
	limit  int
	window time.Duration
	hits   map[string][]time.Time
}

// NewMemoryLimiter returns a limiter allowing limit events per window per key.
func NewMemoryLimiter(limit int, window time.Duration) *MemoryLimiter {
	if limit < 1 {
		limit = 1
	}
	if window <= 0 {
		window = time.Hour
	}

	return &MemoryLimiter{
		limit:  limit,
		window: window,
		hits:   make(map[string][]time.Time),
	}
}

// Allow reports whether key is under the configured rate.
func (l *MemoryLimiter) Allow(key string) bool {
	now := time.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	cutoff := now.Add(-l.window)
	times := l.hits[key]
	kept := times[:0]
	for _, ts := range times {
		if ts.After(cutoff) {
			kept = append(kept, ts)
		}
	}

	if len(kept) >= l.limit {
		l.hits[key] = kept
		return false
	}

	l.hits[key] = append(kept, now)
	return true
}
