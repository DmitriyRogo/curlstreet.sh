package server

import (
	"context"
	"net/http"

	"github.com/sirupsen/logrus"
	"github.com/dmitriy/curlstreet/internal/quote"
	"github.com/dmitriy/curlstreet/internal/service"
)

type QuoteServicer interface {
	FetchQuotes(ctx context.Context, symbols []string, format quote.ResponseFormat) ([]quote.QuoteResult, error)
}

type Server struct {
	svc    QuoteServicer
	mux    *http.ServeMux
	logger *logrus.Logger
}

func New(svc *service.QuoteService, logger *logrus.Logger) *Server {
	s := &Server{svc: svc, mux: http.NewServeMux(), logger: logger}
	s.mux.HandleFunc("/", s.handleQuote)
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) ListenAndServe(addr string, readTimeout, writeTimeout interface{}) error {
	return nil // implemented in main.go via http.Server
}
