package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

type Redis struct {
	cli        *redis.Client
	defaultTTL time.Duration
}

func NewRedis(addr, password string, db int, defaultTTL time.Duration) *Redis {
	cli := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           db,
		MinIdleConns: 1,
		PoolSize:     10,
	})
	return &Redis{cli: cli, defaultTTL: defaultTTL}
}

func (r *Redis) Close() error {
	return r.cli.Close()
}

func (r *Redis) Get(ctx context.Context, key string, v any) (bool, error) {
	b, err := r.cli.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := json.Unmarshal(b, v); err != nil {
		return true, err
	}
	return true, nil
}

func (r *Redis) Set(ctx context.Context, key string, v any, ttl time.Duration) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	exp := ttl
	if exp <= 0 {
		exp = r.defaultTTL
	}
	return r.cli.Set(ctx, key, b, exp).Err()
}

func (r *Redis) Delete(ctx context.Context, key string) error {
	return r.cli.Del(ctx, key).Err()
}
