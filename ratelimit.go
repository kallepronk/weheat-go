package weheat

import (
	"context"
	"sync"
	"time"
)

// Default rate limit. WeHeat caps third-party access at 34 requests per hour per user,
// shared between all clients of that user, so the default leaves headroom.
const (
	DefaultRateLimit  = 30
	DefaultRateWindow = time.Hour
)

// RateLimiter allows at most `limit` calls per sliding `window`. Wait blocks instead of failing.
type RateLimiter struct {
	limit  int
	window time.Duration
	now    func() time.Time

	mu     sync.Mutex
	stamps []time.Time
}

// NewRateLimiter returns a sliding-window limiter.
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{limit: limit, window: window, now: time.Now}
}

// Wait blocks until a request may be made or ctx is done. It records the request on success.
func (r *RateLimiter) Wait(ctx context.Context) error {
	for {
		delay := r.tryAcquire()
		if delay == 0 {
			return nil
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

// tryAcquire records a request and returns 0, or returns how long to wait before retrying.
func (r *RateLimiter) tryAcquire() time.Duration {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := r.now()
	cutoff := now.Add(-r.window)
	kept := r.stamps[:0]
	for _, s := range r.stamps {
		if s.After(cutoff) {
			kept = append(kept, s)
		}
	}
	r.stamps = kept
	if len(r.stamps) < r.limit {
		r.stamps = append(r.stamps, now)
		return 0
	}
	return r.stamps[0].Add(r.window).Sub(now)
}
