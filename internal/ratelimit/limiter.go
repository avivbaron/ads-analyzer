package ratelimit

import (
	"sync"
	"time"
)

type Limiter struct {
	rate      float64  // tokens per second
	burst     float64  // max bucket size
	buckets   sync.Map // key -> *bucket
	now       func() time.Time
	bucketTTL time.Duration // idle eviction
	stopCh    chan struct{}
}

type bucket struct {
	mu       sync.Mutex
	tokens   float64
	last     time.Time // last refill time
	lastSeen time.Time // last Allow call time
}

// New creates a limiter with the given per-second rate and burst.
func New(ratePerSec, burst int) *Limiter {
	if ratePerSec <= 0 {
		ratePerSec = 1
	}
	if burst < ratePerSec {
		burst = ratePerSec
	}
	l := &Limiter{
		rate:      float64(ratePerSec),
		burst:     float64(burst),
		now:       time.Now,
		bucketTTL: 10 * time.Minute,
		stopCh:    make(chan struct{}),
	}
	go l.janitor()
	return l
}

// NewWithClock is for tests to inject a fake clock.
func NewWithClock(ratePerSec, burst int, now func() time.Time) *Limiter {
	l := New(ratePerSec, burst)
	if now != nil {
		l.now = now
	}
	return l
}

func (l *Limiter) Close() { close(l.stopCh) }

// Allow reports whether a request identified by key is permitted now.
// If not allowed, it returns a suggested Retry-After duration.
func (l *Limiter) Allow(key string) (bool, time.Duration) {
	if key == "" {
		key = "_anon"
	}
	bAny, _ := l.buckets.LoadOrStore(key, &bucket{tokens: l.burst, last: l.now(), lastSeen: l.now()})
	b := bAny.(*bucket)

	b.mu.Lock()
	defer b.mu.Unlock()

	now := l.now()
	elapsed := now.Sub(b.last).Seconds()
	if elapsed > 0 {
		b.tokens += l.rate * elapsed
		if b.tokens > l.burst {
			b.tokens = l.burst
		}
		b.last = now
	}

	b.lastSeen = now
	if b.tokens >= 1.0 {
		b.tokens -= 1.0
		return true, 0
	}

	// compute time until we have 1 full token
	need := 1.0 - b.tokens
	secs := need / l.rate
	if secs < 0 {
		secs = 0
	}
	return false, time.Duration(secs * float64(time.Second))
}

func (l *Limiter) janitor() {
	t := time.NewTicker(l.bucketTTL / 2)
	defer t.Stop()
	for {
		select {
		case <-l.stopCh:
			return
		case now := <-t.C:
			l.buckets.Range(func(k, v any) bool {
				b := v.(*bucket)
				b.mu.Lock()
				idle := now.Sub(b.lastSeen)
				b.mu.Unlock()
				if idle > l.bucketTTL {
					l.buckets.Delete(k)
				}
				return true
			})
		}
	}
}
