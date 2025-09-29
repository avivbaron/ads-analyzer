package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/avivbaron/ads-analyzer/internal/config"
)

type Cache interface {
	// Get unmarshals the cached value for key into v.
	// Returns (hit=false, nil) if key is absent or expired.
	Get(ctx context.Context, key string, v any) (bool, error)
	Set(ctx context.Context, key string, v any, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

// NewFromConfig selects a backend based on cfg.CacheBackend.
func NewFromConfig(cfg config.Config) (Cache, func(), error) {
	switch cfg.CacheBackend {
	case "memory":
		mc := NewMemory(MemoryOptions{
			TTL:         cfg.CacheTTL,
			MaxItems:    cfg.CacheMaxItems,
			SweepMin:    cfg.CacheSweepMin,
			SweepMax:    cfg.CacheSweepMax,
			AutoJanitor: true, // production default; tests can turn this off
		})
		return mc, func() { mc.Close() }, nil
	case "redis":
		rc := NewRedis(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB, cfg.CacheTTL)
		closer := func() { _ = rc.Close() }
		return rc, closer, nil	
	default:
		return nil, func() {}, fmt.Errorf("unknown cache backend: %s", cfg.CacheBackend)
	}
}
