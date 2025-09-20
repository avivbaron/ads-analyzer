package httpserver

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/avivbaron/ads-analyzer/internal/metrics"
	"github.com/avivbaron/ads-analyzer/internal/ratelimit"
)

type rlErr struct {
	Error        string `json:"error"`
	RetryAfterMs int64  `json:"retry_after_ms"`
}

func mwRateLimit(lim *ratelimit.Limiter) Middleware {
	return func(next http.Handler) http.Handler {
		if lim == nil {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := clientKey(r)
			allowed, retry := lim.Allow(key)
			if !allowed {
				metrics.IncRateLimited(r.URL.Path) // NEW: record block
				if retry < 0 {
					retry = time.Second
				}
				w.Header().Set("Retry-After", fmt.Sprintf("%d", int(retry.Seconds()+0.999)))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				_ = json.NewEncoder(w).Encode(rlErr{Error: "rate limit exceeded", RetryAfterMs: retry.Milliseconds()})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func clientKey(r *http.Request) string {
	if k := strings.TrimSpace(r.Header.Get("X-API-Key")); k != "" {
		return "k:" + k
	}
	if xf := r.Header.Get("X-Forwarded-For"); xf != "" {
		// take the first IP in the list
		parts := strings.Split(xf, ",")
		return "ip:" + strings.TrimSpace(parts[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return "ip:" + r.RemoteAddr
	}
	return "ip:" + host
}
