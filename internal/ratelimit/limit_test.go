package ratelimit

import (
	"testing"
	"time"
)

// TestLimiter_BasicTokenBucket verifies that the token-bucket limiter:
// - allows up to the burst immediately,
// - denies when empty and returns a positive retry duration,
// - refills at the configured rate over time,
// - isolates buckets by key.
// PASS: first two requests allowed, third denied with retry>0, after 0.5s another request allowed; a different key also allowed.
// FAIL: any of those expectations are not met.
func TestLimiter_BasicTokenBucket(t *testing.T) {
	now := time.Unix(0, 0)
	clock := func() time.Time { return now }

	lim := NewWithClock(2, 2, clock) // 2 tokens/sec, burst 2
	defer lim.Close()

	if ok, _ := lim.Allow("a"); !ok {
		t.Fatalf("want allow 1")
	}
	if ok, _ := lim.Allow("a"); !ok {
		t.Fatalf("want allow 2")
	}
	if ok, retry := lim.Allow("a"); ok || retry <= 0 {
		t.Fatalf("want deny with retry>0, got ok=%v retry=%v", ok, retry)
	}
	now = now.Add(500 * time.Millisecond)
	if ok, _ := lim.Allow("a"); !ok {
		t.Fatalf("want allow after refill")
	}
	if ok, _ := lim.Allow("b"); !ok {
		t.Fatalf("want allow for new key")
	}
}
