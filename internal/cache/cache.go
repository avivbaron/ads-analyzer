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
// For now, only "memory" is implemented.
func NewFromConfig(cfg config.Config) (Cache, func(), error) {
	switch cfg.CacheBackend {
	case "memory":
		mc := NewMemory(cfg.CacheTTL)
		return mc, func() { mc.Close() }, nil
	case "redis":
		rc := NewRedis(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB, cfg.CacheTTL)
		closer := func() { _ = rc.Close() }
		return rc, closer, nil	
	case "file":
		return nil, func() {}, fmt.Errorf("cache backend 'file' not implemented yet")
	default:
		return nil, func() {}, fmt.Errorf("unknown cache backend: %s", cfg.CacheBackend)
	}
}
