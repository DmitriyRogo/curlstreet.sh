package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/dmitriy/curlstreet/internal/quote"
)

type CoinGeckoProvider struct {
	baseURL string
	client  *http.Client
}

func NewCoinGecko(timeout time.Duration) *CoinGeckoProvider {
	return &CoinGeckoProvider{
		baseURL: "https://api.coingecko.com/api/v3",
		client:  &http.Client{Timeout: timeout},
	}
}

func NewCoinGeckoWithBase(baseURL string, timeout time.Duration) *CoinGeckoProvider {
	return &CoinGeckoProvider{
		baseURL: baseURL,
		client:  &http.Client{Timeout: timeout},
	}
}

type coinGeckoMarket struct {
	ID                    string  `json:"id"`
	Symbol                string  `json:"symbol"`
	Name                  string  `json:"name"`
	CurrentPrice          float64 `json:"current_price"`
	PriceChange24h        float64 `json:"price_change_24h"`
	PriceChangePercentage float64 `json:"price_change_percentage_24h"`
	TotalVolume           float64 `json:"total_volume"`
	High24h               float64 `json:"high_24h"`
	Low24h                float64 `json:"low_24h"`
	MarketCap             float64 `json:"market_cap"`
}

func (p *CoinGeckoProvider) Fetch(ctx context.Context, symbol string) (*quote.Quote, error) {
	geckoID, ok := quote.CoinGeckoID(symbol)
	if !ok {
		return nil, ErrSymbolNotFound
	}

	url := fmt.Sprintf("%s/coins/markets?vs_currency=usd&ids=%s", p.baseURL, geckoID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, ErrProviderUnavailable
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, ErrProviderUnavailable
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, ErrProviderUnavailable
	}

	var markets []coinGeckoMarket
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&markets); err != nil {
		return nil, ErrProviderUnavailable
	}
	if len(markets) == 0 {
		return nil, ErrSymbolNotFound
	}

	m := markets[0]
	vol := int64(m.TotalVolume)
	high := m.High24h
	low := m.Low24h
	mcap := int64(m.MarketCap)

	q := &quote.Quote{
		Symbol:        symbol,
		Name:          m.Name,
		Price:         m.CurrentPrice,
		Change:        m.PriceChange24h,
		ChangePercent: m.PriceChangePercentage,
		Volume:        &vol,
		AvgVolume:     nil,
		High52W:       &high,
		Low52W:        &low,
		MarketCap:     &mcap,
		Currency:      "USD",
		MarketStatus:  nil,
		AssetType:     quote.AssetTypeCrypto,
		UpdatedAt:     time.Now().UTC(),
	}
	return q, nil
}
