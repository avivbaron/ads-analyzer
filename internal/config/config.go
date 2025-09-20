package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port         string
	FetchTimeout time.Duration // ads.txt fetch timeout
	HTTPFallback bool          // allow http:// fallback if https fails

	CacheBackend  string        // memory|redis|file (implemented later)
	CacheTTL      time.Duration // TTL for cached results
	RedisAddr     string
	RedisPassword string
	RedisDB       int

	RatePerSec   int
	RateBurst    int
	BatchWorkers int // worker pool size for batch endpoint

	LogLevel          string // info|debug|warn|error
	LogOutput         string // stdout|file|both
	LogFilePath       string // ./logs/ads-analyzer.log
	LogFileMaxSize    int    // MB
	LogFileMaxBackups int    // files
	LogFileMaxAge     int    // days
	LogFileCompress   bool

	MetricsEnabled bool // expose /metrics and collect
}

func Load() (Config, error) {
	c := Config{
		Port:         getenv("PORT", "8080"),
		FetchTimeout: getDurationEnv("FETCH_TIMEOUT", "5s"),
		HTTPFallback: getBoolEnv("HTTP_FALLBACK", true),

		CacheBackend:  strings.ToLower(getenv("CACHE_BACKEND", "memory")),
		CacheTTL:      getDurationEnv("CACHE_TTL", "10m"),
		RedisAddr:     getenv("REDIS_ADDR", "127.0.0.1:6379"),
		RedisPassword: getenv("REDIS_PASSWORD", ""),
		RedisDB:       getIntEnv("REDIS_DB", 0),

		RatePerSec:   getIntEnv("RATE_PER_SEC", 10),
		RateBurst:    getIntEnv("RATE_BURST", 20),
		BatchWorkers: getIntEnv("BATCH_WORKERS", 8),

		LogLevel: strings.ToLower(getenv("LOG_LEVEL", "info")),
		
		LogOutput: strings.ToLower(getenv("LOG_OUTPUT", "stdout")),
		LogFilePath: getenv("LOG_FILE_PATH", "./logs/ads-analyzer.log"),
		LogFileMaxSize: getIntEnv("LOG_FILE_MAX_SIZE", 50),
		LogFileMaxBackups: getIntEnv("LOG_FILE_MAX_BACKUPS", 5),
		LogFileMaxAge: getIntEnv("LOG_FILE_MAX_AGE", 28),
		LogFileCompress: getBoolEnv("LOG_FILE_COMPRESS", true),
	
		MetricsEnabled: getBoolEnv("METRICS_ENABLED", true),
	}

	// Basic sanity checks
	switch c.CacheBackend {
	case "memory", "redis", "file":
	default:
		return Config{}, fmt.Errorf("invalid CACHE_BACKEND: %s", c.CacheBackend)
	}
	if c.RatePerSec <= 0 {
		c.RatePerSec = 1
	}
	if c.RateBurst < c.RatePerSec {
		c.RateBurst = c.RatePerSec
	}
	if c.BatchWorkers <= 0 {
		c.BatchWorkers = 1
	}
	return c, nil
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func getDurationEnv(env, def string) time.Duration {
	if v := os.Getenv(env); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	d, _ := time.ParseDuration(def)
	return d
}

func getBoolEnv(env string, def bool) bool {
	if v := os.Getenv(env); v != "" {
		s := strings.ToLower(v)
		return s == "1" || s == "true" || s == "yes" || s == "y"
	}
	return def
}

func getIntEnv(env string, def int) int {
	if v := os.Getenv(env); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return def
}
