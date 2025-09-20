package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/avivbaron/ads-analyzer/internal/ratelimit"
)

// TestRateLimitMiddleware_Basic checks that a second immediate request is throttled
// when rate=1,burst=1.
// PASS: first request 200, second 429 with Retry-After header.
// FAIL: missing 429 or missing Retry-After.
func TestRateLimitMiddleware_Basic(t *testing.T) {
	lim := ratelimit.New(1, 1)
	defer lim.Close()
	h := mwRateLimit(lim)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }))
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("want 200, got %d", w.Code)
	}
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, r)
	if w2.Code != 429 {
		t.Fatalf("want 429, got %d", w2.Code)
	}
	if w2.Header().Get("Retry-After") == "" {
		t.Fatalf("missing Retry-After")
	}
}
