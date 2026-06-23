package server

import (
	"context"
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

type Server struct {
	svc      QuoteServicer
	calendar CalendarFetcher // nil → static fallback events
	handler  http.Handler
	logger   *logrus.Logger
}

func New(svc *service.QuoteService, logger *logrus.Logger, requestsPerMinute, burst int, calendar CalendarFetcher) *Server {
	mux := http.NewServeMux()
	s := &Server{svc: svc, calendar: calendar, logger: logger}
	mux.HandleFunc("/", s.handleQuote)
	rl := newRateLimiter(requestsPerMinute, burst)
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
