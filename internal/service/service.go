package service

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/sync/errgroup"

	"github.com/DmitriyRogo/curlstreet.sh/internal/provider"
	"github.com/DmitriyRogo/curlstreet.sh/internal/quote"
)

// maxConcurrentFetches bounds how many upstream provider calls a single batch
// may have in flight at once.
const maxConcurrentFetches = 10

type QuoteCache interface {
	Get(symbol string) (*quote.Quote, bool)
	Add(symbol string, q *quote.Quote)
}

type QuoteService struct {
	cache  QuoteCache
	stock  provider.DataProvider
	crypto provider.DataProvider
}

func NewQuoteService(cache QuoteCache, stock, crypto provider.DataProvider) *QuoteService {
	return &QuoteService{cache: cache, stock: stock, crypto: crypto}
}

func (s *QuoteService) FetchQuotes(ctx context.Context, symbols []string, format quote.ResponseFormat) ([]quote.QuoteResult, error) {
	if len(symbols) > 10 {
		return nil, fmt.Errorf("batch limit exceeded. Maximum 10 symbols per request")
	}

	// Resolve each symbol concurrently, writing into a fixed slot so the
	// response order matches the request order. The cache and providers are
	// safe for concurrent use, so no shared-state locking is required here.
	results := make([]quote.QuoteResult, len(symbols))

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(maxConcurrentFetches)
	for i, raw := range symbols {
		g.Go(func() error {
			results[i] = s.fetchOne(ctx, raw)
			return nil
		})
	}
	_ = g.Wait() // fetchOne never returns an error; per-symbol failures live in results.

	return results, nil
}

// fetchOne resolves a single raw symbol into a QuoteResult, mapping provider
// errors to the appropriate SymbolError codes. It never returns an error
// directly so a single bad symbol cannot fail the whole batch.
func (s *QuoteService) fetchOne(ctx context.Context, raw string) quote.QuoteResult {
	if err := quote.ValidateSymbol(raw); err != nil {
		return quote.QuoteResult{
			Err: &quote.SymbolError{
				Symbol:  raw,
				Code:    400,
				Message: fmt.Sprintf("Invalid symbol '%s'. Symbols must be 1–10 alphanumeric characters (hyphens and dots allowed).", raw),
			},
		}
	}

	sym := quote.NormaliseSymbol(raw)

	if q, ok := s.cache.Get(sym); ok {
		q.CacheHit = true
		return quote.QuoteResult{Quote: q}
	}

	prov := s.stock
	if quote.IsCrypto(sym) {
		prov = s.crypto
	}

	q, err := prov.Fetch(ctx, sym)
	if err != nil {
		se := quote.SymbolError{Symbol: sym}
		switch {
		case errors.Is(err, provider.ErrSymbolNotFound):
			se.Code = 404
			se.Message = fmt.Sprintf("Symbol '%s' not found.", sym)
		case errors.Is(err, provider.ErrProviderUnavailable):
			se.Code = 503
			se.Message = "Data source temporarily unavailable. Please try again later."
		case errors.Is(err, context.DeadlineExceeded):
			se.Code = 504
			se.Message = "Request to data source timed out. Please try again later."
		default:
			se.Code = 503
			se.Message = "Data source temporarily unavailable. Please try again later."
		}
		return quote.QuoteResult{Err: &se}
	}

	s.cache.Add(sym, q)
	return quote.QuoteResult{Quote: q}
}
