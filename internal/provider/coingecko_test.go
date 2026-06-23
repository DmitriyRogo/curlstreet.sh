package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCoinGeckoFetch_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]map[string]any{
			{
				"id": "bitcoin", "symbol": "btc", "name": "Bitcoin",
				"current_price": 62000.0, "price_change_24h": 1200.0,
				"price_change_percentage_24h": 1.97,
				"total_volume": 28000000000.0, "high_24h": 63000.0, "low_24h": 61000.0,
			},
		})
	}))
	defer srv.Close()

	p := NewCoinGeckoWithBase(srv.URL, 5*time.Second)
	q, err := p.Fetch(context.Background(), "BTC")

	require.NoError(t, err)
	assert.Equal(t, "BTC", q.Symbol)
	assert.Equal(t, "Bitcoin", q.Name)
	assert.InDelta(t, 62000.0, q.Price, 0.01)
	assert.Nil(t, q.MarketStatus)
	assert.Nil(t, q.AvgVolume)
}

func TestCoinGeckoFetch_EmptyResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]any{})
	}))
	defer srv.Close()

	p := NewCoinGeckoWithBase(srv.URL, 5*time.Second)
	// Use a symbol that exists in crypto_ids but returns empty from API
	// We'll use BTC but mock returns empty
	_, err := p.Fetch(context.Background(), "BTC")
	assert.ErrorIs(t, err, ErrSymbolNotFound)
}

func TestCoinGeckoFetch_UnknownSymbol(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]any{})
	}))
	defer srv.Close()

	p := NewCoinGeckoWithBase(srv.URL, 5*time.Second)
	_, err := p.Fetch(context.Background(), "NOTACRYPTO")
	assert.ErrorIs(t, err, ErrSymbolNotFound)
}

func TestCoinGeckoFetch_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	p := NewCoinGeckoWithBase(srv.URL, 5*time.Second)
	_, err := p.Fetch(context.Background(), "BTC")
	assert.ErrorIs(t, err, ErrProviderUnavailable)
}
