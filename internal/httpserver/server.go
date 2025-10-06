package httpserver

import (
	"context"
	"net/http"
	"time"

	"github.com/avivbaron/ads-analyzer/internal/cache"
	"github.com/avivbaron/ads-analyzer/internal/metrics"
	"github.com/rs/zerolog"

	"github.com/avivbaron/ads-analyzer/internal/ratelimit"
)

type Deps struct {
	Cache        cache.Cache
	Analyzer     Analyzer
	BatchWorkers int
}

type Server struct {
	srv    *http.Server
	logger zerolog.Logger
	deps   Deps
}

type health struct {
	Status string    `json:"status"`
	Time   time.Time `json:"time"`
}

func New(addr string, logger zerolog.Logger, limiter *ratelimit.Limiter, deps Deps, metricsEnabled bool) *Server {
	mux := http.NewServeMux()

	// Register basic service metadata endpoints: health, readiness, and build version
	mux.HandleFunc("/health", HandleHealth())
	mux.HandleFunc("/ready", HandleReady(deps))
	mux.HandleFunc("/version", HandleVersion())

	if metricsEnabled {
		metrics.Init(true)
		mux.Handle("/metrics", metrics.Handler())
	}

	// API routes
	if deps.Analyzer != nil {
		h := NewHandler(deps.Analyzer, deps.BatchWorkers)
		mux.HandleFunc("/api/analysis", h.handleAnalysis)
		mux.HandleFunc("/api/batch-analysis", h.handleBatch)
	}

	// middleware chain
	chain := mwChain(mwRequestID(), mwRateLimit(limiter), mwMetrics(), mwAccessLog(logger))

	s := &http.Server{
		Addr:         addr,
		Handler:      chain(mux),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  10 * time.Second,
	}

	return &Server{srv: s, logger: logger, deps: deps}
}

func (s *Server) Start() error {
	return s.srv.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}
