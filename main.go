package main

import (
	"net/http"

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

	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.WithError(err).Fatal("failed to load config")
	}

	quoteCache, err := cache.New(cfg.Cache.Capacity, cfg.Cache.TTL)
	if err != nil {
		log.WithError(err).Fatal("failed to create cache")
	}

	finnhub := provider.NewFinnhub(cfg.Finnhub.APIKey, cfg.Finnhub.Timeout)
	coinGecko := provider.NewCoinGecko(cfg.CoinGecko.Timeout)

	svc := service.NewQuoteService(quoteCache, finnhub, coinGecko)
	srv := server.New(svc, log)

	httpSrv := &http.Server{
		Addr:         cfg.Server.Addr,
		Handler:      srv,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	log.WithField("addr", cfg.Server.Addr).Info("starting ticker server")
	if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.WithError(err).Fatal("server error")
	}
}
