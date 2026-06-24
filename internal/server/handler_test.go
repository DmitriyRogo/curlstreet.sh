package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/dmitriy/curlstreet/internal/quote"
)

// stubServicer for handler tests
type stubServicer struct {
	results []quote.QuoteResult
	err     error
}

func (s *stubServicer) FetchQuotes(_ context.Context, _ []string, _ quote.ResponseFormat) ([]quote.QuoteResult, error) {
	return s.results, s.err
}

func newTestServer(svc QuoteServicer) *Server {
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)
	mux := http.NewServeMux()
	srv := &Server{svc: svc, logger: logger}
	mux.HandleFunc("/", srv.handleQuote)
	srv.handler = mux
	return srv
}

func makeQuoteResult(sym string, price float64) quote.QuoteResult {
	vol := int64(1000)
	h, l := price * 1.1, price * 0.9
	status := quote.MarketStatusOpen
	return quote.QuoteResult{Quote: &quote.Quote{
		Symbol: sym, Name: "Test", Price: price, Change: 1, ChangePercent: 0.5,
		Volume: &vol, High52W: &h, Low52W: &l,
		Currency: "USD", MarketStatus: &status,
		AssetType: quote.AssetTypeStock, UpdatedAt: time.Now(),
	}}
}

func TestHandleQuote_MethodNotAllowed(t *testing.T) {
	srv := newTestServer(&stubServicer{})
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		req := httptest.NewRequest(method, "/AAPL", nil)
		rr := httptest.NewRecorder()
		srv.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusMethodNotAllowed, rr.Code, "method: %s", method)
	}
}

func TestHandleQuote_Success_PlainText(t *testing.T) {
	result := makeQuoteResult("AAPL", 189.45)
	srv := newTestServer(&stubServicer{results: []quote.QuoteResult{result}})

	req := httptest.NewRequest(http.MethodGet, "/AAPL", nil)
	req.Header.Set("User-Agent", "curl/7.88.0")
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Header().Get("Content-Type"), "text/plain")
	assert.Contains(t, rr.Body.String(), "AAPL")
}

func TestHandleQuote_Success_JSON(t *testing.T) {
	result := makeQuoteResult("AAPL", 189.45)
	srv := newTestServer(&stubServicer{results: []quote.QuoteResult{result}})

	req := httptest.NewRequest(http.MethodGet, "/AAPL?format=json", nil)
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Header().Get("Content-Type"), "application/json")

	var obj map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &obj))
}

func TestHandleQuote_AllErrors_Returns404(t *testing.T) {
	results := []quote.QuoteResult{
		{Err: &quote.SymbolError{Symbol: "XXX", Code: 404, Message: "Symbol 'XXX' not found."}},
	}
	srv := newTestServer(&stubServicer{results: results})

	req := httptest.NewRequest(http.MethodGet, "/XXX", nil)
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestHandleQuote_MixedResults_Returns200(t *testing.T) {
	results := []quote.QuoteResult{
		makeQuoteResult("AAPL", 189.0),
		{Err: &quote.SymbolError{Symbol: "XXX", Code: 404, Message: "not found"}},
	}
	srv := newTestServer(&stubServicer{results: results})

	req := httptest.NewRequest(http.MethodGet, "/AAPL,XXX", nil)
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestHandleQuote_CacheHitHeader(t *testing.T) {
	result := makeQuoteResult("AAPL", 189.45)
	result.Quote.CacheHit = true
	srv := newTestServer(&stubServicer{results: []quote.QuoteResult{result}})

	req := httptest.NewRequest(http.MethodGet, "/AAPL", nil)
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)

	assert.Equal(t, "HIT", rr.Header().Get("X-Cache"))
}

func TestRateLimiter_TrustedProxyUsesXFF(t *testing.T) {
	rl := newRateLimiter(60, 100, "127.0.0.1/32")
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := rl.middleware(inner)

	makeReq := func(xff string) int {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.RemoteAddr = "127.0.0.1:9999"
		r.Header.Set("X-Forwarded-For", xff)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w.Code
	}

	// Two requests from different XFF IPs via the trusted proxy should each
	// be treated as independent clients — neither should be rate-limited.
	assert.Equal(t, http.StatusOK, makeReq("10.0.0.1"))
	assert.Equal(t, http.StatusOK, makeReq("10.0.0.2"))
}

func TestSecurityHeaders(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := securityHeaders(inner)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, "nosniff", rr.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", rr.Header().Get("X-Frame-Options"))
	assert.Equal(t, "no-referrer", rr.Header().Get("Referrer-Policy"))
	assert.NotEmpty(t, rr.Header().Get("Content-Security-Policy"))
}

func TestHandleQuote_CacheMissHeader(t *testing.T) {
	result := makeQuoteResult("AAPL", 189.45)
	result.Quote.CacheHit = false
	srv := newTestServer(&stubServicer{results: []quote.QuoteResult{result}})

	req := httptest.NewRequest(http.MethodGet, "/AAPL", nil)
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)

	assert.Equal(t, "MISS", rr.Header().Get("X-Cache"))
}
