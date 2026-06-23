package server

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/dmitriy/curlstreet/internal/quote"
	"github.com/dmitriy/curlstreet/internal/render"
)

func (s *Server) handleQuote(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		format := quote.DetectFormat(r.Header.Get("User-Agent"), r.URL.Query().Get("format"))
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed. Only GET requests are supported.", format)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" {
		path = r.URL.RawQuery
	}

	symbolsRaw := strings.Split(path, ",")
	symbols := make([]string, 0, len(symbolsRaw))
	for _, s := range symbolsRaw {
		s = strings.TrimSpace(s)
		if s != "" {
			symbols = append(symbols, s)
		}
	}

	format := quote.DetectFormat(r.Header.Get("User-Agent"), r.URL.Query().Get("format"))

	if len(symbols) == 0 {
		writeError(w, http.StatusBadRequest, "No symbol specified.", format)
		return
	}

	results, err := s.svc.FetchQuotes(r.Context(), symbols, format)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error(), format)
		return
	}

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

	// Determine HTTP status
	status := http.StatusOK
	if allErrors(results) && len(results) > 0 {
		status = results[0].Err.Code
	}

	// Set content type
	switch format {
	case quote.ResponseFormatJSON:
		w.Header().Set("Content-Type", "application/json")
	case quote.ResponseFormatHTML:
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	default:
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	}

	body, err := render.Render(format, results)
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
