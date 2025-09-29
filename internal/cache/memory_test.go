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

func TestMemory_SetGet(t *testing.T) {
	mc := NewMemory(MemoryOptions{
		TTL:         0,
		MaxItems:    0,
		SweepMin:    time.Second,
		SweepMax:    time.Minute,
		AutoJanitor: false,
		Now:         time.Now,
	})
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

func TestMemory_TTLExpiry(t *testing.T) {
	base := time.Unix(0, 0)
	now := base
	clock := func() time.Time { return now }

	mc := NewMemory(MemoryOptions{
		TTL:         100 * time.Millisecond,
		MaxItems:    0,
		SweepMin:    time.Second,
		SweepMax:    time.Minute,
		AutoJanitor: false, // we’ll sweep manually
		Now:         clock,
	})
	defer mc.Close()

	ctx := context.Background()
	_ = mc.Set(ctx, "k", sample{A: "y", B: 1}, 0)

	var out sample
	hit, _ := mc.Get(ctx, "k", &out)
	if !hit {
		t.Fatalf("want initial hit")
	}

	// Advance time past TTL and sweep
	now = now.Add(200 * time.Millisecond)
	mc.sweepOnce()

	hit, _ = mc.Get(ctx, "k", &out)
	if hit {
		t.Fatalf("expected miss after TTL expiry")
	}
}

// LRU: inserting beyond cap evicts least-recently used key.
func TestMemory_LRU_EvictOrder(t *testing.T) {
	mc := NewMemory(MemoryOptions{
		TTL:         0,
		MaxItems:    3,
		SweepMin:    time.Second,
		SweepMax:    time.Minute,
		AutoJanitor: false,
		Now:         time.Now,
	})
	defer mc.Close()
	ctx := context.Background()

	set := func(k string, v int) {
		if err := mc.Set(ctx, k, sample{A: "v", B: v}, 0); err != nil {
			t.Fatalf("set %s: %v", k, err)
		}
	}
	get := func(k string) bool {
		var out sample
		hit, err := mc.Get(ctx, k, &out)
		if err != nil {
			t.Fatalf("get %s: %v", k, err)
		}
		return hit
	}

	// Fill to capacity.
	set("A", 1)
	set("B", 2)
	set("C", 3)

	// Access order now: C (MRU), B, A (LRU) — because last insert is MRU.
	// Insert D -> should evict A.
	set("D", 4)

	if get("A") {
		t.Fatalf("expected A to be evicted")
	}
	if !get("B") || !get("C") || !get("D") {
		t.Fatalf("expected B,C,D present")
	}
}

// LRU promotion on Get: touching a key should protect it from the next eviction.
func TestMemory_LRU_PromoteOnGet(t *testing.T) {
	mc := NewMemory(MemoryOptions{
		TTL:         0,
		MaxItems:    2,
		SweepMin:    time.Second,
		SweepMax:    time.Minute,
		AutoJanitor: false,
		Now:         time.Now,
	})
	defer mc.Close()
	ctx := context.Background()

	_ = mc.Set(ctx, "X", sample{A: "x", B: 1}, 0)
	_ = mc.Set(ctx, "Y", sample{A: "y", B: 2}, 0)

	// Promote X
	var tmp sample
	hit, err := mc.Get(ctx, "X", &tmp)
	if err != nil || !hit {
		t.Fatalf("get X: hit=%v err=%v", hit, err)
	}

	// Insert Z -> should evict Y (LRU), not X.
	_ = mc.Set(ctx, "Z", sample{A: "z", B: 3}, 0)

	hit, _ = mc.Get(ctx, "Y", &tmp)
	if hit {
		t.Fatalf("expected Y evicted")
	}
	if hit, _ = mc.Get(ctx, "X", &tmp); !hit {
		t.Fatalf("expected X present")
	}
	if hit, _ = mc.Get(ctx, "Z", &tmp); !hit {
		t.Fatalf("expected Z present")
	}
}

// TTL beats LRU: expired entries are removed regardless of recency.
func TestMemory_LRU_TTLBeatsLRU(t *testing.T) {
	base := time.Unix(0, 0)
	now := base
	clock := func() time.Time { return now }

	mc := NewMemory(MemoryOptions{
		TTL:         50 * time.Millisecond,
		MaxItems:    2,
		SweepMin:    time.Second,
		SweepMax:    time.Minute,
		AutoJanitor: false,
		Now:         clock,
	})
	defer mc.Close()
	ctx := context.Background()

	_ = mc.Set(ctx, "K1", sample{A: "k1", B: 1}, 0)
	_ = mc.Set(ctx, "K2", sample{A: "k2", B: 2}, 0)

	// Touch both to promote recency
	var tmp sample
	_, _ = mc.Get(ctx, "K1", &tmp)
	_, _ = mc.Get(ctx, "K2", &tmp)

	// Advance past TTL and sweep
	now = now.Add(60 * time.Millisecond)
	mc.sweepOnce()

	// Both should be gone due to TTL expiry despite being MRU recently.
	if hit, _ := mc.Get(ctx, "K1", &tmp); hit {
		t.Fatalf("expected K1 expired")
	}
	if hit, _ := mc.Get(ctx, "K2", &tmp); hit {
		t.Fatalf("expected K2 expired")
	}
}

// Interval caps: with tiny TTL, sweep uses lower clamp; with huge TTL, upper clamp.
// We can't observe ticker timing deterministically here, but we can assert clamp math by
// indirectly checking chosen path via sweep behavior + manual Now() control.
func TestMemory_SweepManual_NoAutoJanitor(t *testing.T) {
	// This test mainly validates that manual sweep works deterministically.
	base := time.Unix(0, 0)
	now := base
	clock := func() time.Time { return now }

	mc := NewMemory(MemoryOptions{
		TTL:         10 * time.Millisecond,
		MaxItems:    10,
		SweepMin:    time.Second,      // would clamp TTL/2 up to 1s
		SweepMax:    30 * time.Second, // upper cap
		AutoJanitor: false,
		Now:         clock,
	})
	defer mc.Close()
	ctx := context.Background()

	_ = mc.Set(ctx, "S", sample{A: "s", B: 1}, 0)

	var out sample
	if hit, _ := mc.Get(ctx, "S", &out); !hit {
		t.Fatalf("expected hit before expiry")
	}
	now = now.Add(20 * time.Millisecond) // > TTL
	mc.sweepOnce()
	if hit, _ := mc.Get(ctx, "S", &out); hit {
		t.Fatalf("expected miss after manual sweep past TTL")
	}
}
