package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"github.com/dmitriy/curlstreet/internal/cache"
	"github.com/dmitriy/curlstreet/internal/config"
	"github.com/dmitriy/curlstreet/internal/provider"
	"github.com/dmitriy/curlstreet/internal/server"
	"github.com/dmitriy/curlstreet/internal/service"
)

func main() {
	log := logrus.New()
	log.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})

	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		log.WithError(err).Warn("failed to parse .env file")
	}

	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.WithError(err).Fatal("failed to load config")
	}

	quoteCache, err := cache.New(cfg.Cache.Capacity, cfg.Cache.TTL)
	if err != nil {
		log.WithError(err).Fatal("failed to create cache")
	}

	log.WithField("finnhub_key_set", cfg.Finnhub.APIKey != "").Info("provider config")

	finnhub := provider.NewFinnhub(cfg.Finnhub.APIKey, cfg.Finnhub.Timeout)
	coinGecko := provider.NewCoinGecko(cfg.CoinGecko.Timeout)

	// Startup connectivity probe — logs whether Finnhub is reachable so
	// deployment issues show up immediately in the startup log.
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		_, err := finnhub.Fetch(ctx, "AAPL")
		if err != nil {
			log.WithError(err).Warn("finnhub connectivity probe failed")
		} else {
			log.Info("finnhub connectivity probe OK")
		}
	}()

	svc := service.NewQuoteService(quoteCache, finnhub, coinGecko)
	srv := server.New(svc, log, cfg.RateLimit.RequestsPerMinute, cfg.RateLimit.Burst, cfg.Server.TrustedProxy, server.ComputedCalendar(), finnhub)

	httpSrv := &http.Server{
		Addr:         cfg.Server.Addr,
		Handler:      srv,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	go func() {
		log.WithField("addr", cfg.Server.Addr).Info("starting ticker server")
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.WithError(err).Fatal("server error")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down server")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(ctx); err != nil {
		log.WithError(err).Fatal("server forced to shutdown")
	}
	log.Info("server stopped")
}
