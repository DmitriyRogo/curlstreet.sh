package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/dmitriy/curlstreet/internal/provider"
	"github.com/dmitriy/curlstreet/internal/quote"
)

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

	results := make([]quote.QuoteResult, 0, len(symbols))

	for _, raw := range symbols {
		if err := quote.ValidateSymbol(raw); err != nil {
			results = append(results, quote.QuoteResult{
				Err: &quote.SymbolError{
					Symbol:  raw,
					Code:    400,
					Message: fmt.Sprintf("Invalid symbol '%s'. Symbols must be 1–10 alphanumeric characters (hyphens and dots allowed).", raw),
				},
			})
			continue
		}

		sym := quote.NormaliseSymbol(raw)

		if q, ok := s.cache.Get(sym); ok {
			q.CacheHit = true
			results = append(results, quote.QuoteResult{Quote: q})
			continue
		}

		var prov provider.DataProvider
		if quote.IsCrypto(sym) {
			prov = s.crypto
		} else {
			prov = s.stock
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
			results = append(results, quote.QuoteResult{Err: &se})
			continue
		}

		s.cache.Add(sym, q)
		results = append(results, quote.QuoteResult{Quote: q})
	}

	return results, nil
}
