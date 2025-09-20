package analysis

import (
	"context"
	"testing"
	"time"

	"github.com/avivbaron/ads-analyzer/internal/cache"
)

type fakeFetcher struct {
	calls int
	data  []byte
	err   error
}

func (f *fakeFetcher) GetAdsTxt(ctx context.Context, domain string) ([]byte, error) {
	f.calls++
	return f.data, f.err
}

// TestService_Analyze_CachesResult verifies that first Analyze fetches & parses,
// and subsequent Analyze for the same domain returns a cached response without calling fetcher.
// PASS: first call not cached; second call Cached=true and fetcher called only once.
// FAIL: second call misses cache or fetcher invoked again.
func TestService_Analyze_CachesResult(t *testing.T) {
	ctx := context.Background()
	mc := cache.NewMemoryWithClock(1*time.Minute, time.Now)
	defer mc.Close()
	ff := &fakeFetcher{data: []byte("google.com, x, DIRECT\nappnexus.com, x, DIRECT\ngoogle.com, y, RESELLER\n")}
	svc := NewService(mc, ff, 1*time.Minute)
	res1, err := svc.Analyze(ctx, "msn.com")
	if err != nil {
		t.Fatalf("analyze1 err: %v", err)
	}
	if ff.calls != 1 {
		t.Fatalf("fetcher calls=%d want 1", ff.calls)
	}
	if res1.Cached {
		t.Fatalf("first call should not be cached")
	}
	res2, err := svc.Analyze(ctx, "https://msn.com/ads.txt")
	if err != nil {
		t.Fatalf("analyze2 err: %v", err)
	}
	if !res2.Cached {
		t.Fatalf("second call expected cached=true")
	}
	if ff.calls != 1 {
		t.Fatalf("fetcher should not be called again; calls=%d", ff.calls)
	}
}
