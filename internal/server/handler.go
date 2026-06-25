package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/DmitriyRogo/curlstreet.sh/internal/quote"
	"github.com/DmitriyRogo/curlstreet.sh/internal/render"
)

var marketIndices = []struct {
	symbol string
	name   string
}{
	{"SPY", "S&P 500 ETF Trust"},
	{"QQQ", "Nasdaq 100 ETF"},
	{"DIA", "Dow Jones ETF"},
}

// indexNames provides fallback display names for ETF/index symbols
// that Finnhub's profile2 endpoint returns empty for.
var indexNames = map[string]string{
	"SPY": "S&P 500 ETF Trust",
	"QQQ": "Nasdaq 100 ETF",
	"DIA": "Dow Jones ETF",
}

func (s *Server) handleQuote(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		format := quote.DetectFormat(r.Header.Get("User-Agent"), r.URL.Query().Get("format"))
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed. Only GET requests are supported.", format)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/")
	format := quote.DetectFormat(r.Header.Get("User-Agent"), r.URL.Query().Get("format"))

	// Kick off the market-index banner concurrently with the user's ticker
	// fetch so it never serially adds to response latency. It runs under its
	// own bounded context (defer cancels it on any early return) and degrades
	// to "(market data unavailable)" rather than risking the write deadline.
	marketCh := make(chan []quote.QuoteResult, 1)
	marketCtx, marketCancel := context.WithTimeout(r.Context(), s.marketOverviewTimeout())
	defer marketCancel()
	if format != quote.ResponseFormatJSON {
		go func() { marketCh <- s.fetchMarketOverview(marketCtx, format) }()
	} else {
		marketCh <- nil
	}

	var tickerResults []quote.QuoteResult
	if path != "" {
		symbolsRaw := strings.Split(path, ",")
		symbols := make([]string, 0, len(symbolsRaw))
		for _, sym := range symbolsRaw {
			sym = strings.TrimSpace(sym)
			if sym != "" {
				symbols = append(symbols, sym)
			}
		}
		if len(symbols) > 0 {
			var err error
			tickerResults, err = s.svc.FetchQuotes(r.Context(), symbols, format)
			if err != nil {
				writeError(w, http.StatusBadRequest, err.Error(), format)
				return
			}
		}
	}

	marketResults := <-marketCh

	// Fetch live economic calendar for the homepage when not in JSON mode.
	var econEvents []quote.EconEvent
	if path == "" && format != quote.ResponseFormatJSON && s.calendar != nil {
		econEvents, _ = s.calendar.FetchEconomicCalendar(r.Context())
	}

	results := append(marketResults, tickerResults...)

	// Determine X-Cache header
	allHit := true
	for _, qr := range results {
		if qr.Quote == nil || !qr.Quote.CacheHit {
			allHit = false
			break
		}
	}
	if allHit && len(results) > 0 {
		w.Header().Set("X-Cache", "HIT")
	} else {
		w.Header().Set("X-Cache", "MISS")
	}

	// HTTP status driven by user-requested ticker errors only
	status := http.StatusOK
	if allErrors(tickerResults) && len(tickerResults) > 0 {
		status = tickerResults[0].Err.Code
	}

	switch format {
	case quote.ResponseFormatJSON:
		w.Header().Set("Content-Type", "application/json")
	case quote.ResponseFormatHTML:
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	default:
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	}

	body, err := render.Render(format, results, econEvents...)
	if err != nil {
		s.logger.WithError(err).Error("render error")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(status)
	fmt.Fprint(w, body)
}

func (s *Server) marketOverviewTimeout() time.Duration {
	if s.marketTimeout > 0 {
		return s.marketTimeout
	}
	return defaultMarketOverviewTimeout
}

// fetchMarketOverview fetches the market-index banner quotes and tags them for
// rendering. It returns nil when no index quote could be retrieved (timeout or
// upstream failure) so the banner degrades to "(market data unavailable)"
// instead of showing a row of errors.
func (s *Server) fetchMarketOverview(ctx context.Context, format quote.ResponseFormat) []quote.QuoteResult {
	syms := make([]string, len(marketIndices))
	for i, idx := range marketIndices {
		syms[i] = idx.symbol
	}

	results, _ := s.svc.FetchQuotes(ctx, syms, format)

	hasQuote := false
	for i := range results {
		results[i].IsMarket = true
		if results[i].Quote == nil {
			continue
		}
		hasQuote = true
		// Fill in display name for ETFs whose profile2 returns empty.
		if results[i].Quote.Name == "" {
			if name, ok := indexNames[results[i].Quote.Symbol]; ok {
				results[i].Quote.Name = name
			}
		}
	}

	if !hasQuote {
		return nil
	}
	return results
}

func allErrors(results []quote.QuoteResult) bool {
	for _, qr := range results {
		if qr.Quote != nil {
			return false
		}
	}
	return true
}

func writeError(w http.ResponseWriter, code int, msg string, format quote.ResponseFormat) {
	switch format {
	case quote.ResponseFormatJSON:
		w.Header().Set("Content-Type", "application/json")
	case quote.ResponseFormatHTML:
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	default:
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	}
	w.WriteHeader(code)
	fmt.Fprint(w, render.RenderError(code, msg, format))
}
