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
	srv := &Server{svc: svc, mux: http.NewServeMux(), logger: logger}
	srv.mux.HandleFunc("/", srv.handleQuote)
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

func TestHandleQuote_CacheMissHeader(t *testing.T) {
	result := makeQuoteResult("AAPL", 189.45)
	result.Quote.CacheHit = false
	srv := newTestServer(&stubServicer{results: []quote.QuoteResult{result}})

	req := httptest.NewRequest(http.MethodGet, "/AAPL", nil)
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)

	assert.Equal(t, "MISS", rr.Header().Get("X-Cache"))
}
