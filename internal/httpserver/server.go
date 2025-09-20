package httpserver

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"time"

	"github.com/avivbaron/ads-analyzer/internal/buildinfo"
	"github.com/avivbaron/ads-analyzer/internal/cache"
	"github.com/avivbaron/ads-analyzer/internal/logs"
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

	// Health
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(health{Status: "ok", Time: time.Now().UTC()})
	})

	// Readiness: verify cache roundtrip quickly
	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if deps.Cache == nil {
			_ = json.NewEncoder(w).Encode(health{Status: "ready", Time: time.Now().UTC()})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 200*time.Millisecond)
		defer cancel()
		type probe struct {
			OK string `json:"ok"`
		}
		key := "ready:" + time.Now().Format("20060102150405.000")
		p := probe{OK: "1"}
		if err := deps.Cache.Set(ctx, key, p, time.Second); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "not ready", "error": err.Error()})
			return
		}
		var out probe
		hit, err := deps.Cache.Get(ctx, key, &out)
		_ = deps.Cache.Delete(context.Background(), key)
		if err != nil || !hit || out.OK != "1" {
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "not ready"})
			return
		}
		_ = json.NewEncoder(w).Encode(health{Status: "ready", Time: time.Now().UTC()})
	})

	// buildinfo(values injeceted at build time)
	mux.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"version":    buildinfo.Version,
			"commit":     buildinfo.Commit,
			"build_time": buildinfo.BuildTime,
			"go":         buildinfo.Go,
		})
	})

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

	s := &http.Server{
		Addr:         addr,
		Handler:      mwChain(mwRequestID(), mwRateLimit(limiter), mwMetrics(), mwAccessLog(logger))(mux),
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

// mw = middleware
type Middleware func(http.Handler) http.Handler

// mwChain applies each middleware in declaration order so the earliest one wraps the handler last.\r\nfunc mwChain(mwFuncs ...middleware) middleware {
func mwChain(mwFuncs ...Middleware) Middleware {
	return func(next http.Handler) http.Handler {
		for i := len(mwFuncs) - 1; i >= 0; i-- {
			next = mwFuncs[i](next)
		}
		return next
	}
}

// mwRequestID attaches a short-lived request identifier to the context and outgoing headers.
func mwRequestID() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := shortID()
			r = r.WithContext(withReqID(r.Context(), id))
			w.Header().Set("X-Request-ID", id)
			next.ServeHTTP(w, r)
		})
	}
}

// mwAccessLog records request and response details to the provided logger.
func mwAccessLog(logger zerolog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rl := &logs.RespLogger{ResponseWriter: w, Status: 200}
			next.ServeHTTP(rl, r)

			clientIP, _, _ := net.SplitHostPort(r.RemoteAddr)
			if xf := r.Header.Get("X-Forwarded-For"); xf != "" {
				clientIP = xf
			}

			logger.Info().
				Str("id", reqID(r.Context())).
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Int("status", rl.Status).
				Int("bytes", rl.Bytes).
				Str("ip", clientIP).
				Dur("dur", time.Since(start)).
				Msg("http")
		})
	}
}
