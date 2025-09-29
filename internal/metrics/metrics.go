package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	Registry        *prometheus.Registry
	HTTPRequests    *prometheus.CounterVec
	HTTPDuration    *prometheus.HistogramVec
	CacheHits       *prometheus.CounterVec
	CacheMisses     *prometheus.CounterVec
	FetchDuration   *prometheus.HistogramVec
	RateLimitBlocks *prometheus.CounterVec
}

var M *Metrics

func Init(enabled bool) *Metrics {
	if !enabled {
		M = nil
		return nil
	}
	r := prometheus.NewRegistry()
	m := &Metrics{
		Registry:        r,
		HTTPRequests:    prometheus.NewCounterVec(prometheus.CounterOpts{Name: "http_requests_total", Help: "HTTP requests total"}, []string{"method", "path", "status"}),
		HTTPDuration:    prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "http_request_duration_seconds", Help: "HTTP request duration", Buckets: prometheus.DefBuckets}, []string{"method", "path", "status"}),
		CacheHits:       prometheus.NewCounterVec(prometheus.CounterOpts{Name: "cache_hits_total", Help: "Cache hits"}, []string{"op"}),
		CacheMisses:     prometheus.NewCounterVec(prometheus.CounterOpts{Name: "cache_misses_total", Help: "Cache misses"}, []string{"op"}),
		FetchDuration:   prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "fetch_duration_seconds", Help: "ads.txt fetch duration", Buckets: prometheus.DefBuckets}, []string{"scheme"}),
		RateLimitBlocks: prometheus.NewCounterVec(prometheus.CounterOpts{Name: "rate_limit_blocks_total", Help: "Requests blocked by rate limiter"}, []string{"path"}), // NEW
	}
	r.MustRegister(m.HTTPRequests, m.HTTPDuration, m.CacheHits, m.CacheMisses, m.FetchDuration, m.RateLimitBlocks)
	M = m
	return m
}

func Handler() http.Handler {
	return promhttp.HandlerFor(M.Registry, promhttp.HandlerOpts{})
}

// helpers (safe no-ops if M==nil)
func IncHit(op string) {
	if M != nil {
		M.CacheHits.WithLabelValues(op).Inc()
	}
}

func IncMiss(op string) {
	if M != nil {
		M.CacheMisses.WithLabelValues(op).Inc()
	}
}

func ObserveFetch(scheme string, start time.Time) {
	if M != nil {
		M.FetchDuration.WithLabelValues(scheme).Observe(time.Since(start).Seconds())
	}
}

func IncRateLimited(path string) {
	if M != nil {
		M.RateLimitBlocks.WithLabelValues(path).Inc()
	}
}
