package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
	"github.com/dmitriy/curlstreet/internal/quote"
)

type stubCache struct{}

func (s *stubCache) Get(_ string) (*quote.Quote, bool) { return nil, false }
func (s *stubCache) Add(_ string, _ *quote.Quote)      {}

type stubProvider struct {
	q   *quote.Quote
	err error
}

func (s *stubProvider) Fetch(_ context.Context, sym string) (*quote.Quote, error) {
	if s.err != nil {
		return nil, s.err
	}
	q := *s.q
	q.Symbol = sym
	return &q, nil
}

func makeStockQuote() *quote.Quote {
	vol := int64(1000000)
	h, l := 200.0, 150.0
	status := quote.MarketStatusOpen
	return &quote.Quote{
		Symbol: "X", Name: "Test", Price: 100, Change: 1, ChangePercent: 1,
		Volume: &vol, High52W: &h, Low52W: &l, Currency: "USD",
		MarketStatus: &status, AssetType: quote.AssetTypeStock, UpdatedAt: time.Now(),
	}
}

// P10: Batch completeness — exactly M quotes + (N-M) errors for N-symbol batch
func TestBatchCompleteness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(1, 10).Draw(t, "n")
		m := rapid.IntRange(0, n).Draw(t, "m")

		validSyms := make([]string, m)
		for i := range validSyms {
			validSyms[i] = fmt.Sprintf("S%d", i+1)
		}
		invalidSyms := make([]string, n-m)
		for i := range invalidSyms {
			invalidSyms[i] = fmt.Sprintf("BAD SYM%d", i)
		}

		symbols := append(validSyms, invalidSyms...)

		prov := &stubProvider{q: makeStockQuote()}
		svc := NewQuoteService(&stubCache{}, prov, prov)

		results, err := svc.FetchQuotes(context.Background(), symbols, quote.ResponseFormatText)
		require.NoError(t, err)
		assert.Len(t, results, n)

		quotes := 0
		errs := 0
		for _, r := range results {
			if r.Quote != nil {
				quotes++
			} else {
				errs++
			}
		}
		assert.Equal(t, m, quotes)
		assert.Equal(t, n-m, errs)
	})
}

func TestBatchLimitExceeded(t *testing.T) {
	prov := &stubProvider{q: makeStockQuote()}
	svc := NewQuoteService(&stubCache{}, prov, prov)
	symbols := make([]string, 11)
	for i := range symbols {
		symbols[i] = fmt.Sprintf("S%d", i)
	}
	_, err := svc.FetchQuotes(context.Background(), symbols, quote.ResponseFormatText)
	assert.Error(t, err)
}
