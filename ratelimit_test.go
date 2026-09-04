package weheat

import (
	"context"
	"errors"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func staticToken(tok string) oauth2.TokenSource {
	return oauth2.StaticTokenSource(&oauth2.Token{AccessToken: tok})
}

func TestRateLimiterBlocksThenFrees(t *testing.T) {
	now := time.Unix(0, 0)
	r := NewRateLimiter(2, time.Hour)
	r.now = func() time.Time { return now }

	for i := 0; i < 2; i++ {
		if r.tryAcquire() != 0 {
			t.Fatalf("call %d should pass", i)
		}
	}
	if d := r.tryAcquire(); d != time.Hour {
		t.Fatalf("third should wait 1h, got %v", d)
	}
	now = now.Add(time.Hour + time.Second)
	if r.tryAcquire() != 0 {
		t.Fatal("should pass after window")
	}
}

func TestRateLimiterWaitHonoursContext(t *testing.T) {
	r := NewRateLimiter(1, time.Hour)
	_ = r.Wait(context.Background())
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if err := r.Wait(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("want deadline exceeded, got %v", err)
	}
}
