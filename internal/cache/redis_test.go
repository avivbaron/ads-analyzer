package cache

import (
	"context"
	"os"
	"testing"
	"time"
)

type payload struct {
	A string
	B int
}

// TestRedis_SetGetTTL verifies Redis-backed cache roundtrip and TTL expiry.
// PASS: get after set hits and equals; after TTL, get misses. Skips if Redis not reachable.
// FAIL: when Redis is reachable but roundtrip/expiry behavior incorrect.
func TestRedis_SetGetTTL(t *testing.T) {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "127.0.0.1:6379"
	}
	r := NewRedis(addr, os.Getenv("REDIS_PASSWORD"), 0, 100*time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := r.cli.Ping(ctx).Err(); err != nil {
		t.Skipf("skipping: redis not reachable at %s: %v", addr, err)
	}
	defer r.Close()
	key := "test:redis:" + time.Now().Format("150405.000")
	in := payload{A: "x", B: 7}
	if err := r.Set(ctx, key, in, 50*time.Millisecond); err != nil {
		t.Fatalf("set: %v", err)
	}
	var out payload
	hit, err := r.Get(ctx, key, &out)
	if err != nil || !hit {
		t.Fatalf("get1 err=%v hit=%v", err, hit)
	}
	if out != in {
		t.Fatalf("roundtrip mismatch: %#v vs %#v", out, in)
	}
	time.Sleep(80 * time.Millisecond)
	hit, err = r.Get(ctx, key, &out)
	if err != nil {
		t.Fatalf("get2 err=%v", err)
	}
	if hit {
		t.Fatalf("expected miss after TTL")
	}
}
