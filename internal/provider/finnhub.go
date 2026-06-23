package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/dmitriy/curlstreet/internal/quote"
)

type FinnhubProvider struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

func NewFinnhub(apiKey string, timeout time.Duration) *FinnhubProvider {
	return &FinnhubProvider{
		apiKey:  apiKey,
		baseURL: "https://finnhub.io/api/v1",
		client:  &http.Client{Timeout: timeout},
	}
}

func NewFinnhubWithBase(apiKey, baseURL string, timeout time.Duration) *FinnhubProvider {
	return &FinnhubProvider{
		apiKey:  apiKey,
		baseURL: baseURL,
		client:  &http.Client{Timeout: timeout},
	}
}

type finnhubQuoteResp struct {
	C  float64 `json:"c"`
	D  float64 `json:"d"`
	Dp float64 `json:"dp"`
	H  float64 `json:"h"`
	L  float64 `json:"l"`
}

type finnhubProfileResp struct {
	Name     string `json:"name"`
	Currency string `json:"currency"`
}

func (p *FinnhubProvider) Fetch(ctx context.Context, symbol string) (*quote.Quote, error) {
	qr, err := p.fetchQuote(ctx, symbol)
	if err != nil {
		return nil, err
	}
	if qr.C == 0 && qr.D == 0 {
		return nil, ErrSymbolNotFound
	}

	pr, err := p.fetchProfile(ctx, symbol)
	if err != nil {
		return nil, err
	}

	currency := pr.Currency
	if currency == "" {
		currency = "USD"
	}

	h, l := qr.H, qr.L
	status := quote.MarketStatusClosed

	vol := int64(0)
	q := &quote.Quote{
		Symbol:        symbol,
		Name:          pr.Name,
		Price:         qr.C,
		Change:        qr.D,
		ChangePercent: qr.Dp,
		Volume:        &vol,
		High52W:       &h,
		Low52W:        &l,
		Currency:      currency,
		MarketStatus:  &status,
		AssetType:     quote.AssetTypeStock,
		UpdatedAt:     time.Now().UTC(),
	}
	return q, nil
}

func (p *FinnhubProvider) fetchQuote(ctx context.Context, symbol string) (*finnhubQuoteResp, error) {
	url := fmt.Sprintf("%s/quote?symbol=%s&token=%s", p.baseURL, symbol, p.apiKey)
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

	var r finnhubQuoteResp
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, ErrProviderUnavailable
	}
	return &r, nil
}

func (p *FinnhubProvider) fetchProfile(ctx context.Context, symbol string) (*finnhubProfileResp, error) {
	url := fmt.Sprintf("%s/stock/profile2?symbol=%s&token=%s", p.baseURL, symbol, p.apiKey)
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

	var r finnhubProfileResp
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, ErrProviderUnavailable
	}
	return &r, nil
}
