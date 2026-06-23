package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mockFinnhubServer(quoteBody, profileBody, marketStatusBody map[string]any) *httptest.Server {
	return mockFinnhubServerWithMetric(quoteBody, profileBody, marketStatusBody,
		map[string]any{"metric": map[string]any{"52WeekHigh": 200.0, "52WeekLow": 150.0}})
}

func mockFinnhubServerWithMetric(quoteBody, profileBody, marketStatusBody, metricBody map[string]any) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/quote", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(quoteBody)
	})
	mux.HandleFunc("/stock/profile2", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(profileBody)
	})
	mux.HandleFunc("/stock/market-status", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(marketStatusBody)
	})
	mux.HandleFunc("/stock/metric", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(metricBody)
	})
	return httptest.NewServer(mux)
}

func TestFinnhubFetch_Success(t *testing.T) {
	srv := mockFinnhubServer(
		map[string]any{"c": 189.45, "d": 1.23, "dp": 0.65, "h": 191.0, "l": 187.5},
		map[string]any{"name": "Apple Inc.", "currency": "USD", "mic": "XNAS"},
		map[string]any{"isOpen": true, "session": "regular"},
	)
	defer srv.Close()

	p := NewFinnhubWithBase("testkey", srv.URL, 5*time.Second)
	q, err := p.Fetch(context.Background(), "AAPL")

	require.NoError(t, err)
	assert.Equal(t, "AAPL", q.Symbol)
	assert.Equal(t, "Apple Inc.", q.Name)
	assert.InDelta(t, 189.45, q.Price, 0.001)
	assert.InDelta(t, 1.23, q.Change, 0.001)
	require.NotNil(t, q.MarketStatus)
	assert.Equal(t, "OPEN", *q.MarketStatus)
}

func TestFinnhubFetch_MarketClosed(t *testing.T) {
	srv := mockFinnhubServer(
		map[string]any{"c": 189.45, "d": 1.23, "dp": 0.65, "h": 191.0, "l": 187.5},
		map[string]any{"name": "Apple Inc.", "currency": "USD", "mic": "XNAS"},
		map[string]any{"isOpen": false, "session": ""},
	)
	defer srv.Close()

	p := NewFinnhubWithBase("testkey", srv.URL, 5*time.Second)
	q, err := p.Fetch(context.Background(), "AAPL")

	require.NoError(t, err)
	require.NotNil(t, q.MarketStatus)
	assert.Equal(t, "CLOSED", *q.MarketStatus)
}

func TestFinnhubFetch_PreMarket(t *testing.T) {
	srv := mockFinnhubServer(
		map[string]any{"c": 189.45, "d": 1.23, "dp": 0.65, "h": 191.0, "l": 187.5},
		map[string]any{"name": "Apple Inc.", "currency": "USD", "mic": "XNAS"},
		map[string]any{"isOpen": true, "session": "pre_market"},
	)
	defer srv.Close()

	p := NewFinnhubWithBase("testkey", srv.URL, 5*time.Second)
	q, err := p.Fetch(context.Background(), "AAPL")

	require.NoError(t, err)
	require.NotNil(t, q.MarketStatus)
	assert.Equal(t, "PRE_MARKET", *q.MarketStatus)
}

func TestFinnhubFetch_AfterHours(t *testing.T) {
	srv := mockFinnhubServer(
		map[string]any{"c": 189.45, "d": 1.23, "dp": 0.65, "h": 191.0, "l": 187.5},
		map[string]any{"name": "Apple Inc.", "currency": "USD", "mic": "XNAS"},
		map[string]any{"isOpen": true, "session": "after_market"},
	)
	defer srv.Close()

	p := NewFinnhubWithBase("testkey", srv.URL, 5*time.Second)
	q, err := p.Fetch(context.Background(), "AAPL")

	require.NoError(t, err)
	require.NotNil(t, q.MarketStatus)
	assert.Equal(t, "AFTER_HOURS", *q.MarketStatus)
}

func TestFinnhubFetch_MarketStatusFallbackOnError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/quote", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"c": 189.45, "d": 1.23, "dp": 0.65, "h": 191.0, "l": 187.5, "v": 50000000.0})
	})
	mux.HandleFunc("/stock/profile2", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"name": "Apple Inc.", "currency": "USD"})
	})
	mux.HandleFunc("/stock/market-status", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	mux.HandleFunc("/stock/metric", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"metric": map[string]any{"52WeekHigh": 200.0, "52WeekLow": 150.0}})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	p := NewFinnhubWithBase("testkey", srv.URL, 5*time.Second)
	q, err := p.Fetch(context.Background(), "AAPL")

	require.NoError(t, err)
	require.NotNil(t, q.MarketStatus)
	assert.Equal(t, "CLOSED", *q.MarketStatus)
}

func TestFinnhubFetch_VolumeAnd52W(t *testing.T) {
	srv := mockFinnhubServerWithMetric(
		map[string]any{"c": 189.45, "d": 1.23, "dp": 0.65, "h": 191.0, "l": 187.5, "v": 55123456.0},
		map[string]any{"name": "Apple Inc.", "currency": "USD", "mic": "XNAS"},
		map[string]any{"isOpen": true, "session": "regular"},
		map[string]any{"metric": map[string]any{"52WeekHigh": 199.62, "52WeekLow": 124.17}},
	)
	defer srv.Close()

	p := NewFinnhubWithBase("testkey", srv.URL, 5*time.Second)
	q, err := p.Fetch(context.Background(), "AAPL")

	require.NoError(t, err)
	require.NotNil(t, q.Volume)
	assert.Equal(t, int64(55123456), *q.Volume)
	require.NotNil(t, q.High52W)
	assert.InDelta(t, 199.62, *q.High52W, 0.001)
	require.NotNil(t, q.Low52W)
	assert.InDelta(t, 124.17, *q.Low52W, 0.001)
}

func TestFinnhubFetch_52WFallsBackToIntradayOnMetricError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/quote", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"c": 189.45, "d": 1.23, "dp": 0.65, "h": 191.0, "l": 187.5})
	})
	mux.HandleFunc("/stock/profile2", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"name": "Apple Inc.", "currency": "USD"})
	})
	mux.HandleFunc("/stock/market-status", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"isOpen": false, "session": nil})
	})
	mux.HandleFunc("/stock/metric", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	p := NewFinnhubWithBase("testkey", srv.URL, 5*time.Second)
	q, err := p.Fetch(context.Background(), "AAPL")

	require.NoError(t, err)
	require.NotNil(t, q.High52W)
	assert.InDelta(t, 191.0, *q.High52W, 0.001) // intraday fallback
	assert.InDelta(t, 187.5, *q.Low52W, 0.001)
}

func TestFinnhubFetch_SymbolNotFound(t *testing.T) {
	srv := mockFinnhubServer(
		map[string]any{"c": 0, "d": 0, "dp": 0},
		map[string]any{},
		map[string]any{"isOpen": false, "session": ""},
	)
	defer srv.Close()

	p := NewFinnhubWithBase("testkey", srv.URL, 5*time.Second)
	_, err := p.Fetch(context.Background(), "XXXX")

	assert.ErrorIs(t, err, ErrSymbolNotFound)
}

func TestFinnhubFetch_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	p := NewFinnhubWithBase("testkey", srv.URL, 5*time.Second)
	_, err := p.Fetch(context.Background(), "AAPL")

	assert.ErrorIs(t, err, ErrProviderUnavailable)
}

func TestFinnhubProvider_OversizedBody(t *testing.T) {
	huge := strings.Repeat("x", 2<<20) // 2 MiB of garbage
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, huge)
	}))
	defer srv.Close()

	p := NewFinnhubWithBase("testkey", srv.URL, 5*time.Second)
	_, err := p.Fetch(context.Background(), "AAPL")
	require.Error(t, err, "oversized body should return an error")
}

func TestFinnhubFetch_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// hang forever
		<-r.Context().Done()
	}))
	defer srv.Close()

	p := NewFinnhubWithBase("testkey", srv.URL, 50*time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := p.Fetch(ctx, "AAPL")
	assert.Error(t, err)
}
