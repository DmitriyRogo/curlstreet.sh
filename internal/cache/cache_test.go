package cache

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
	"github.com/DmitriyRogo/curlstreet.sh/internal/quote"
)

func makeQuote(symbol string) *quote.Quote {
	return &quote.Quote{Symbol: symbol, Name: "Test", Price: 100}
}

// P8: Cache TTL enforcement — HIT before expiry, MISS at/after expiry
func TestCacheTTLEnforcement(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ttl := time.Duration(rapid.Int64Range(1, 1000).Draw(t, "ttl_ms")) * time.Millisecond
		now := time.Now()
		clock := now

		c, err := NewWithClock(100, ttl, func() time.Time { return clock })
		require.NoError(t, err)

		q := makeQuote("AAPL")
		c.Add("AAPL", q)

		// HIT before expiry
		clock = now.Add(ttl - time.Nanosecond)
		got, ok := c.Get("AAPL")
		assert.True(t, ok, "expected HIT before expiry")
		assert.Equal(t, q, got)

		// MISS after expiry (expiresAt.Before uses strict less-than)
		clock = now.Add(ttl).Add(time.Nanosecond)
		_, ok = c.Get("AAPL")
		assert.False(t, ok, "expected MISS after expiry")
	})
}

// Get must return an isolated copy: mutating it (as the service does with
// CacheHit, or the handler does with Name) must not corrupt the cached entry
// or any other reader.
func TestCacheGetReturnsIsolatedCopy(t *testing.T) {
	c, err := New(10, time.Minute)
	require.NoError(t, err)
	c.Add("AAPL", makeQuote("AAPL"))

	first, ok := c.Get("AAPL")
	require.True(t, ok)
	first.CacheHit = true
	first.Name = "mutated"

	second, ok := c.Get("AAPL")
	require.True(t, ok)
	assert.False(t, second.CacheHit, "CacheHit must not leak across Get calls")
	assert.Equal(t, "Test", second.Name, "Name mutation must not leak into the cache")
	assert.NotSame(t, first, second, "each Get must return a distinct pointer")
}

// Add must store a copy so the caller mutating its own reference after Add
// does not reach into the cached entry.
func TestCacheAddIsolatesCallerReference(t *testing.T) {
	c, err := New(10, time.Minute)
	require.NoError(t, err)

	q := makeQuote("AAPL")
	c.Add("AAPL", q)
	q.Name = "mutated after add"

	got, ok := c.Get("AAPL")
	require.True(t, ok)
	assert.Equal(t, "Test", got.Name, "post-Add mutation must not reach the cache")
}

// Concurrent readers mutating their own copies must be race-free under -race.
func TestCacheConcurrentReadersAreRaceSafe(t *testing.T) {
	c, err := New(10, time.Minute)
	require.NoError(t, err)
	c.Add("AAPL", makeQuote("AAPL"))

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			q, ok := c.Get("AAPL")
			if ok {
				q.CacheHit = true
				q.Name = fmt.Sprintf("req-%d", n)
			}
		}(i)
	}
	wg.Wait()
}

// P9: Cache symbol isolation — invalidating A doesn't affect B
func TestCacheSymbolIsolation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		c, err := New(100, time.Minute)
		require.NoError(t, err)

		symA := rapid.StringMatching(`[A-Z]{1,5}`).Draw(t, "symA")
		symB := rapid.StringMatching(`[A-Z]{1,5}`).Draw(t, "symB")
		if symA == symB {
			return
		}

		qA := makeQuote(symA)
		qB := makeQuote(symB)
		c.Add(symA, qA)
		c.Add(symB, qB)

		c.Remove(symA)

		_, ok := c.Get(symA)
		assert.False(t, ok, "A should be gone after Remove")

		got, ok := c.Get(symB)
		assert.True(t, ok, "B should still be present")
		assert.Equal(t, qB, got)
	})
}
