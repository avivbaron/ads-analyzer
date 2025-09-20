package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/avivbaron/ads-analyzer/internal/analysis"
	"github.com/avivbaron/ads-analyzer/internal/cache"
	"github.com/avivbaron/ads-analyzer/internal/config"
	"github.com/avivbaron/ads-analyzer/internal/httpserver"
	"github.com/avivbaron/ads-analyzer/internal/logs"
	"github.com/avivbaron/ads-analyzer/internal/ratelimit"
)

func loadDotenv() {
    // Load .env if it exists, but don't fail if it's missing.
    if _, err := os.Stat(".env"); err == nil {
        if err := godotenv.Load(); err != nil {
            log.Printf("warning: couldn't load .env: %v", err)
        }
    }
}

func main() {
	// Load .env file
	loadDotenv()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	// init logger
	opts := logs.Options{
		Level:          cfg.LogLevel,
		Output:         cfg.LogOutput,
		FilePath:       cfg.LogFilePath,
		FileMaxSizeMB:  cfg.LogFileMaxSize,
		FileMaxBackups: cfg.LogFileMaxBackups,
		FileMaxAgeDays: cfg.LogFileMaxAge,
		FileCompress:   cfg.LogFileCompress,
	}
	logger := logs.NewWithOptions(opts)

	// init rate limiter
	limiter := ratelimit.New(cfg.RatePerSec, cfg.RateBurst)
	defer limiter.Close()

	// init cache
	c, closeCache, err := cache.NewFromConfig(cfg)
	if err != nil {
		logger.Fatal().Err(err).Msg("cache init failed")
	}
	defer closeCache()

	fetcher := analysis.NewHTTPFetcher(cfg.FetchTimeout, cfg.HTTPFallback)
	svc := analysis.NewService(c, fetcher, cfg.CacheTTL)

	addr := ":" + cfg.Port
	serverDeps := httpserver.Deps{
		Cache:        c,
		Analyzer:     svc,
		BatchWorkers: cfg.BatchWorkers,
	}
	srv := httpserver.New(addr, logger, limiter, serverDeps, cfg.MetricsEnabled)

	// start server
	go func() {
		logger.Info().Str("addr", addr).Int("rate_per_sec", cfg.RatePerSec).Int("burst", cfg.RateBurst).Msg("listening")
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			logger.Info().Str("addr", addr).Msg("listening")
		}
	}()

	// gracefull shutdown
	stop := make(chan os.Signal, 1)
	// os.Interrupt is a closing signal from Ctrl+C.
	// syscall.SIGTERM is a closing signal from Docker/k8s.
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop // waiting for close signal

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error().Err(err).Msg("graceful shutdown error")
	}
	logger.Info().Msg("server stopped")

}
