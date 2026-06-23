package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Addr         string        `yaml:"addr"`
		ReadTimeout  time.Duration `yaml:"read_timeout"`
		WriteTimeout time.Duration `yaml:"write_timeout"`
	} `yaml:"server"`
	Cache struct {
		Capacity int           `yaml:"capacity"`
		TTL      time.Duration `yaml:"ttl"`
	} `yaml:"cache"`
	Finnhub struct {
		APIKey  string        `yaml:"api_key"`
		Timeout time.Duration `yaml:"timeout"`
	} `yaml:"finnhub"`
	CoinGecko struct {
		Timeout time.Duration `yaml:"timeout"`
	} `yaml:"coingecko"`
}

func Load(path string) (*Config, error) {
	cfg := &Config{}
	cfg.Server.Addr = ":8080"
	cfg.Server.ReadTimeout = 10 * time.Second
	cfg.Server.WriteTimeout = 10 * time.Second
	cfg.Cache.Capacity = 1000
	cfg.Cache.TTL = 60 * time.Second
	cfg.Finnhub.Timeout = 5 * time.Second
	cfg.CoinGecko.Timeout = 5 * time.Second

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}
	defer f.Close()

	if err := yaml.NewDecoder(f).Decode(cfg); err != nil {
		return nil, err
	}

	if key := os.Getenv("FINNHUB_API_KEY"); key != "" {
		cfg.Finnhub.APIKey = key
	}

	return cfg, nil
}
