package httpserver

import (
	"net"
	"net/http"
	"time"

	"github.com/avivbaron/ads-analyzer/internal/logs"
	"github.com/rs/zerolog"
)

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
