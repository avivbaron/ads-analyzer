package cache

import (
	"container/list"
	"context"
	"encoding/json"
	"sync"
	"time"
)

type MemoryOptions struct {
	TTL         time.Duration
	MaxItems    int           // 0 => unlimited
	SweepMin    time.Duration // clamp lower bound for janitor tick
	SweepMax    time.Duration // clamp upper bound for janitor tick
	AutoJanitor bool          // start the janitor goroutine
	Now         func() time.Time
}

type Memory struct {
	mu       sync.RWMutex
	m        map[string]*entry
	lru      *list.List // most-recent at Front(), least-recent at Back()
	maxItems int        // 0 => unlimited
	ttl      time.Duration
	sweepMin time.Duration
	sweepMax time.Duration
	now      func() time.Time
	stop     chan struct{}
}

type entry struct {
	data []byte
	exp  time.Time     // zero => no expiry
	el   *list.Element // points into lru; nil if unlinked
}

func NewMemory(opt MemoryOptions) *Memory {
	if opt.Now == nil {
		opt.Now = time.Now
	}
	if opt.SweepMin <= 0 {
		opt.SweepMin = time.Second
	}
	if opt.SweepMax < opt.SweepMin {
		opt.SweepMax = opt.SweepMin
	}
	mc := &Memory{
		m:        make(map[string]*entry),
		lru:      list.New(),
		maxItems: opt.MaxItems,
		ttl:      opt.TTL,
		sweepMin: opt.SweepMin,
		sweepMax: opt.SweepMax,
		now:      opt.Now,
		stop:     make(chan struct{}),
	}
	if opt.AutoJanitor {
		go mc.janitor()
	}
	return mc
}

func (mc *Memory) Close() {
	close(mc.stop)
}

func (mc *Memory) Get(ctx context.Context, key string, v any) (bool, error) {
	mc.mu.RLock()
	e, ok := mc.m[key]
	if !ok {
		mc.mu.RUnlock()
		return false, nil
	} else if !e.exp.IsZero() && mc.now().After(e.exp) {
		mc.mu.RUnlock()
		_ = mc.Delete(ctx, key)
		return false, nil
	}

	// Copy bytes while under read lock, then unlock for JSON decode
	data := make([]byte, len(e.data))
	copy(data, e.data)
	mc.mu.RUnlock()

	// Promote recency under write lock
	mc.mu.Lock()
	if e.el != nil {
		mc.lru.MoveToFront(e.el)
	}
	mc.mu.Unlock()

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
	defer mc.mu.Unlock()
	if e, ok := mc.m[key]; ok {
		e.data = b
		e.exp = exp
		if e.el != nil {
			mc.lru.MoveToFront(e.el) // promote
		}
		return nil
	}

	el := mc.lru.PushFront(key)
	mc.m[key] = &entry{data: b, exp: exp, el: el}

	// Enforce cap
	if mc.maxItems > 0 && mc.lru.Len() > mc.maxItems {
		mc.evictLRU()
	}

	return nil
}

func (mc *Memory) Delete(ctx context.Context, key string) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	if e, ok := mc.m[key]; ok {
		if e.el != nil {
			mc.lru.Remove(e.el)
			e.el = nil
		}
		delete(mc.m, key)
	}
	return nil
}

func (mc *Memory) evictLRU() {
	for mc.maxItems > 0 && mc.lru.Len() > mc.maxItems {
		back := mc.lru.Back()
		if back == nil {
			return
		}
		k := back.Value.(string)
		mc.lru.Remove(back)
		if e, ok := mc.m[k]; ok {
			e.el = nil
			delete(mc.m, k)
		}
	}
}

func (mc *Memory) sweepOnce() {
    now := mc.now()
    mc.mu.Lock()
    for k, e := range mc.m {
        if !e.exp.IsZero() && now.After(e.exp) {
            if e.el != nil {
                mc.lru.Remove(e.el)
                e.el = nil
            }
            delete(mc.m, k)
        }
    }
    mc.mu.Unlock()
}

func (mc *Memory) janitor() {
    var interval time.Duration
    if mc.ttl <= 0 {
        interval = mc.sweepMax // no TTL => lazy sweep
    } else {
        interval = mc.ttl/2
    }
	
    t := time.NewTicker(interval)
    defer t.Stop()

    for {
        select {
        case <-mc.stop:
            return
        case <-t.C:
            mc.sweepOnce()
        }
    }
}
