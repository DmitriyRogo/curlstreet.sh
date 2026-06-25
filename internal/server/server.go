package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/dmitriy/curlstreet/internal/quote"
	"github.com/dmitriy/curlstreet/internal/service"
)

type QuoteServicer interface {
	FetchQuotes(ctx context.Context, symbols []string, format quote.ResponseFormat) ([]quote.QuoteResult, error)
}

// CalendarFetcher retrieves upcoming economic events.
type CalendarFetcher interface {
	FetchEconomicCalendar(ctx context.Context) ([]quote.EconEvent, error)
}

// Prober tests external connectivity (e.g. Finnhub reachability).
type Prober interface {
	Probe(ctx context.Context) error
}

type Server struct {
	svc      QuoteServicer
	calendar CalendarFetcher // nil → static fallback events
	prober   Prober          // nil → health check skips provider probe
	handler  http.Handler
	logger   *logrus.Logger
}

func New(svc *service.QuoteService, logger *logrus.Logger, requestsPerMinute, burst int, trustedProxy string, calendar CalendarFetcher, prober ...Prober) *Server {
	mux := http.NewServeMux()
	s := &Server{svc: svc, calendar: calendar, logger: logger}
	if len(prober) > 0 {
		s.prober = prober[0]
	}
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/", s.handleQuote)
	rl := newRateLimiter(requestsPerMinute, burst, trustedProxy)
	s.handler = securityHeaders(rl.middleware(mux))
	return s
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy",
			"default-src 'none'; style-src 'unsafe-inline' https://fonts.googleapis.com; font-src https://fonts.gstatic.com; img-src data:")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.handler.ServeHTTP(w, r)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if s.prober == nil {
		fmt.Fprintln(w, "ok")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	if err := s.prober.Probe(ctx); err != nil {
		s.logger.WithError(err).Warn("finnhub probe failed")
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintln(w, "finnhub unavailable")
		return
	}
	fmt.Fprintln(w, "ok")
}

func (s *Server) ListenAndServe(addr string, readTimeout, writeTimeout interface{}) error {
	return nil // implemented in main.go via http.Server
}

// ComputedCalendar returns a CalendarFetcher backed by the built-in US
// economic event schedule — no external API key required.
func ComputedCalendar() CalendarFetcher { return &computedCalendar{} }

type computedCalendar struct{}

func (c *computedCalendar) FetchEconomicCalendar(_ context.Context) ([]quote.EconEvent, error) {
	return quote.UpcomingEconEvents(time.Now()), nil
}
