package httpserver

import (
	"fmt"
	"net/http"
	"time"

	"github.com/avivbaron/ads-analyzer/internal/logs"
	"github.com/avivbaron/ads-analyzer/internal/metrics"
)

func mwMetrics() Middleware {
	return func(next http.Handler) http.Handler {
		if metrics.M == nil {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rl := &logs.RespLogger{ResponseWriter: w, Status: 200}
			next.ServeHTTP(rl, r)
			status := rl.Status
			labels := []string{r.Method, r.URL.Path, fmt.Sprintf("%d", status)}
			metrics.M.HTTPRequests.WithLabelValues(labels...).Inc()
			metrics.M.HTTPDuration.WithLabelValues(labels...).Observe(time.Since(start).Seconds())
		})
	}
}
