package cache

import (
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/dmitriy/curlstreet/internal/quote"
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

func (c *QuoteCache) Get(symbol string) (*quote.Quote, bool) {
	e, ok := c.lru.Get(symbol)
	if !ok {
		return nil, false
	}
	if e.expiresAt.Before(c.timeFn()) {
		c.lru.Remove(symbol)
		return nil, false
	}
	return e.quote, true
}

func (c *QuoteCache) Add(symbol string, q *quote.Quote) {
	c.lru.Add(symbol, cacheEntry{
		quote:     q,
		expiresAt: c.timeFn().Add(c.ttl),
	})
}

func (c *QuoteCache) Remove(symbol string) {
	c.lru.Remove(symbol)
}
