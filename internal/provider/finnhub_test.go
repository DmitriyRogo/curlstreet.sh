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

func TestFinnhubFetch_Success(t *testing.T) {
	quoteCalled := false
	profileCalled := false

	mux := http.NewServeMux()
	mux.HandleFunc("/quote", func(w http.ResponseWriter, r *http.Request) {
		quoteCalled = true
		json.NewEncoder(w).Encode(map[string]any{
			"c": 189.45, "d": 1.23, "dp": 0.65, "h": 191.0, "l": 187.5,
		})
	})
	mux.HandleFunc("/stock/profile2", func(w http.ResponseWriter, r *http.Request) {
		profileCalled = true
		json.NewEncoder(w).Encode(map[string]any{
			"name": "Apple Inc.", "currency": "USD",
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	p := NewFinnhubWithBase("testkey", srv.URL, 5*time.Second)
	q, err := p.Fetch(context.Background(), "AAPL")

	require.NoError(t, err)
	assert.True(t, quoteCalled)
	assert.True(t, profileCalled)
	assert.Equal(t, "AAPL", q.Symbol)
	assert.Equal(t, "Apple Inc.", q.Name)
	assert.InDelta(t, 189.45, q.Price, 0.001)
	assert.InDelta(t, 1.23, q.Change, 0.001)
}

func TestFinnhubFetch_SymbolNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/quote", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"c": 0, "d": 0, "dp": 0})
	})
	mux.HandleFunc("/stock/profile2", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{})
	})

	srv := httptest.NewServer(mux)
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
