package server

import (
	"fmt"
	"net/http"
	"strings"

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

	// Fetch market index overview for text/HTML formats (not JSON)
	var marketResults []quote.QuoteResult
	if format != quote.ResponseFormatJSON {
		syms := make([]string, len(marketIndices))
		for i, idx := range marketIndices {
			syms[i] = idx.symbol
		}
		marketResults, _ = s.svc.FetchQuotes(r.Context(), syms, format)
		for i := range marketResults {
			marketResults[i].IsMarket = true
			// Fill in display name for ETFs whose profile2 returns empty
			if marketResults[i].Quote != nil && marketResults[i].Quote.Name == "" {
				if name, ok := indexNames[marketResults[i].Quote.Symbol]; ok {
					marketResults[i].Quote.Name = name
				}
			}
		}
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
