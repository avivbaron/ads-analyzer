package cache

import (
	"context"
	"encoding/json"
	"sync"
	"time"
)

type Memory struct {
	mu   sync.RWMutex
	m    map[string]entry
	ttl  time.Duration
	now  func() time.Time
	stop chan struct{}
}

type entry struct {
	data []byte
	exp  time.Time // zero => no expiry
}

func NewMemory(ttl time.Duration) *Memory { return NewMemoryWithClock(ttl, time.Now) }

// NewMemoryWithClock lets tests inject a fake clock.
func NewMemoryWithClock(ttl time.Duration, now func() time.Time) *Memory {
	mc := &Memory{
		m:    make(map[string]entry),
		ttl:  ttl,
		now:  now,
		stop: make(chan struct{}),
	}
	go mc.janitor()
	return mc
}

func (mc *Memory) Close() { close(mc.stop) }

func (mc *Memory) Get(ctx context.Context, key string, v any) (bool, error) {
	mc.mu.RLock()
	e, ok := mc.m[key]
	mc.mu.RUnlock()
	if !ok {
		return false, nil
	}
	if !e.exp.IsZero() && mc.now().After(e.exp) {
		mc.mu.Lock()
		delete(mc.m, key)
		mc.mu.Unlock()
		return false, nil
	}
	if err := json.Unmarshal(e.data, v); err != nil {
		return true, err
	}
	return true, nil
}

func (mc *Memory) Set(ctx context.Context, key string, v any, ttl time.Duration) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	var exp time.Time
	t := mc.ttl
	if ttl > 0 {
		t = ttl
	}
	if t > 0 {
		exp = mc.now().Add(t)
	}
	mc.mu.Lock()
	mc.m[key] = entry{data: b, exp: exp}
	mc.mu.Unlock()
	return nil
}

func (mc *Memory) Delete(ctx context.Context, key string) error {
	mc.mu.Lock()
	delete(mc.m, key)
	mc.mu.Unlock()
	return nil
}

func (mc *Memory) janitor() {
	interval := mc.ttl / 2
	if interval <= 0 {
		interval = time.Minute
	}
	if interval < time.Second {
		interval = time.Second
	}
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-mc.stop:
			return
		case <-t.C:
			now := mc.now()
			mc.mu.Lock()
			for k, e := range mc.m {
				if !e.exp.IsZero() && now.After(e.exp) {
					delete(mc.m, k)
				}
			}
			mc.mu.Unlock()
		}
	}
}
