package cache

import (
	"context"
	"testing"
	"time"
)

type sample struct {
	A string
	B int
}

// TestMemory_SetGet ensures values can be JSON-encoded, stored, and decoded back.
// PASS: Get returns hit=true and decoded struct equals original.
// FAIL: hit=false or decoded value mismatches.
func TestMemory_SetGet(t *testing.T) {
	mc := NewMemoryWithClock(0, time.Now)
	defer mc.Close()
	ctx := context.Background()
	in := sample{A: "x", B: 42}
	if err := mc.Set(ctx, "k1", in, 0); err != nil {
		t.Fatalf("set: %v", err)
	}
	var out sample
	hit, err := mc.Get(ctx, "k1", &out)
	if err != nil {
		t.Fatalf("get err: %v", err)
	}
	if !hit {
		t.Fatalf("expected hit")
	}
	if out != in {
		t.Fatalf("roundtrip mismatch: %#v vs %#v", out, in)
	}
}

// TestMemory_TTLExpiry verifies that entries expire after TTL and become misses.
// PASS: first Get is a hit, after advancing beyond TTL the next Get is a miss.
// FAIL: second Get still returns a hit.
func TestMemory_TTLExpiry(t *testing.T) {
	base := time.Unix(0, 0)
	now := base
	clock := func() time.Time { return now }
	mc := NewMemoryWithClock(100*time.Millisecond, clock)
	defer mc.Close()
	ctx := context.Background()
	_ = mc.Set(ctx, "k", sample{A: "y", B: 1}, 0)
	var out sample
	hit, _ := mc.Get(ctx, "k", &out)
	if !hit {
		t.Fatalf("want initial hit")
	}
	now = now.Add(200 * time.Millisecond)
	hit, _ = mc.Get(ctx, "k", &out)
	if hit {
		t.Fatalf("expected miss after TTL expiry")
	}
}
