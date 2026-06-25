package cache

import (
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/DmitriyRogo/curlstreet.sh/internal/quote"
)

type cacheEntry struct {
	quote     *quote.Quote
	expiresAt time.Time
}

type QuoteCache struct {
	lru    *lru.Cache[string, cacheEntry]
	ttl    time.Duration
	timeFn func() time.Time
}

func New(capacity int, ttl time.Duration) (*QuoteCache, error) {
	l, err := lru.New[string, cacheEntry](capacity)
	if err != nil {
		return nil, err
	}
	return &QuoteCache{lru: l, ttl: ttl, timeFn: time.Now}, nil
}

func NewWithClock(capacity int, ttl time.Duration, timeFn func() time.Time) (*QuoteCache, error) {
	l, err := lru.New[string, cacheEntry](capacity)
	if err != nil {
		return nil, err
	}
	return &QuoteCache{lru: l, ttl: ttl, timeFn: timeFn}, nil
}

// Get returns an isolated copy of the cached quote. Callers may freely mutate
// the returned value (e.g. set CacheHit or fill a display name) without
// affecting the cached entry or any other concurrent caller.
func (c *QuoteCache) Get(symbol string) (*quote.Quote, bool) {
	e, ok := c.lru.Get(symbol)
	if !ok {
		return nil, false
	}
	if e.expiresAt.Before(c.timeFn()) {
		c.lru.Remove(symbol)
		return nil, false
	}
	return cloneQuote(e.quote), true
}

// Add stores an isolated copy of q so the caller retaining its own reference
// cannot later mutate the cached entry.
func (c *QuoteCache) Add(symbol string, q *quote.Quote) {
	c.lru.Add(symbol, cacheEntry{
		quote:     cloneQuote(q),
		expiresAt: c.timeFn().Add(c.ttl),
	})
}

// cloneQuote returns a shallow copy of q. Pointer fields (e.g. High52W,
// MarketStatus) are shared by reference and must be treated as immutable —
// callers replace them rather than writing through them.
func cloneQuote(q *quote.Quote) *quote.Quote {
	if q == nil {
		return nil
	}
	cp := *q
	return &cp
}

func (c *QuoteCache) Remove(symbol string) {
	c.lru.Remove(symbol)
}
