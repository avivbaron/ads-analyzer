package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/avivbaron/ads-analyzer/internal/metrics"
	"github.com/avivbaron/ads-analyzer/internal/ratelimit"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

// TestRateLimit_Metrics increments the rate_limit_blocks_total counter when a request is blocked.
// PASS: counter for this path == 1 after a blocked call.
// FAIL: counter remains 0.
func TestRateLimit_Metrics(t *testing.T) {
	metrics.Init(true)
	lim := ratelimit.New(1, 1)
	defer lim.Close()

	h := mwRateLimit(lim)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }))

	r := httptest.NewRequest(http.MethodGet, "/metricspath", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r) // consumes burst
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, r) // blocked

	c := testutil.ToFloat64(metrics.M.RateLimitBlocks.WithLabelValues("/metricspath"))
	if c < 1 {
		t.Fatalf("expected rate_limit_blocks_total >= 1, got %v", c)
	}
}
