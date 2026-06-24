package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/dmitriy/curlstreet/internal/quote"
)

type marketStatusEntry struct {
	status    string
	expiresAt time.Time
}

type FinnhubProvider struct {
	apiKey  string
	baseURL string
	client  *http.Client

	msMu    sync.Mutex
	msCache map[string]marketStatusEntry
	sf      singleflight.Group
}

func NewFinnhub(apiKey string, timeout time.Duration) *FinnhubProvider {
	return &FinnhubProvider{
		apiKey:  apiKey,
		baseURL: "https://finnhub.io/api/v1",
		client:  &http.Client{Timeout: timeout},
		msCache: make(map[string]marketStatusEntry),
	}
}

func NewFinnhubWithBase(apiKey, baseURL string, timeout time.Duration) *FinnhubProvider {
	return &FinnhubProvider{
		apiKey:  apiKey,
		baseURL: baseURL,
		client:  &http.Client{Timeout: timeout},
		msCache: make(map[string]marketStatusEntry),
	}
}

type finnhubQuoteResp struct {
	C  float64 `json:"c"`
	D  float64 `json:"d"`
	Dp float64 `json:"dp"`
	H  float64 `json:"h"`
	L  float64 `json:"l"`
	V  float64 `json:"v"`
}

type finnhubMetricResp struct {
	Metric struct {
		High52W   float64 `json:"52WeekHigh"`
		Low52W    float64 `json:"52WeekLow"`
		MarketCap float64 `json:"marketCapitalization"` // in millions USD
	} `json:"metric"`
}

type finnhubMarketStatusResp struct {
	IsOpen  bool   `json:"isOpen"`
	Session string `json:"session"`
}

type finnhubProfileResp struct {
	Name            string `json:"name"`
	Currency        string `json:"currency"`
	Exchange        string `json:"exchange"`
	Mic             string `json:"mic"`
	Type            string `json:"type"`
	FinnhubIndustry string `json:"finnhubIndustry"`
	GsubInd         string `json:"gsubInd"`
}

type fetchResult[T any] struct {
	val T
	err error
}

func (p *FinnhubProvider) Fetch(ctx context.Context, symbol string) (*quote.Quote, error) {
	qCh := make(chan fetchResult[*finnhubQuoteResp], 1)
	pCh := make(chan fetchResult[*finnhubProfileResp], 1)
	mCh := make(chan fetchResult[*finnhubMetricResp], 1)

	go func() { v, e := p.fetchQuote(ctx, symbol); qCh <- fetchResult[*finnhubQuoteResp]{v, e} }()
	go func() { v, e := p.fetchProfile(ctx, symbol); pCh <- fetchResult[*finnhubProfileResp]{v, e} }()
	go func() { v, e := p.fetchMetric(ctx, symbol); mCh <- fetchResult[*finnhubMetricResp]{v, e} }()

	qRes, pRes, mRes := <-qCh, <-pCh, <-mCh

	if qRes.err != nil {
		return nil, qRes.err
	}
	if pRes.err != nil {
		return nil, pRes.err
	}
	qr, pr := qRes.val, pRes.val

	if qr.C == 0 && qr.D == 0 {
		return nil, ErrSymbolNotFound
	}

	currency := pr.Currency
	if currency == "" {
		currency = "USD"
	}

	status := p.marketStatus(ctx, "US")

	// Use intraday high/low as fallback if 52W metric is unavailable.
	high52, low52 := qr.H, qr.L
	var mcap *int64
	if mRes.err == nil && mRes.val.Metric.High52W > 0 {
		high52 = mRes.val.Metric.High52W
		low52 = mRes.val.Metric.Low52W
	}
	if mRes.err == nil && mRes.val.Metric.MarketCap > 0 {
		v := int64(mRes.val.Metric.MarketCap * 1_000_000)
		mcap = &v
	}

	q := &quote.Quote{
		Symbol:        symbol,
		Name:          pr.Name,
		Price:         qr.C,
		Change:        qr.D,
		ChangePercent: qr.Dp,
		Volume:        nil,
		MarketCap:     mcap,
		High52W:       &high52,
		Low52W:        &low52,
		Currency:      currency,
		MarketStatus:  &status,
		AssetType:     quote.AssetTypeStock,
		Exchange:      exchangeShortName(pr.Mic, pr.Exchange),
		ExchangeCode:  pr.Mic,
		SecurityType:  pr.Type,
		Sector:        pr.FinnhubIndustry,
		Industry:      pr.GsubInd,
		UpdatedAt:     time.Now().UTC(),
	}
	return q, nil
}

// marketStatus returns the current market status for the given Finnhub
// exchange code, using a 60-second in-process cache to avoid a round-trip
// on every quote request.
func (p *FinnhubProvider) marketStatus(ctx context.Context, exchange string) string {
	p.msMu.Lock()
	if e, ok := p.msCache[exchange]; ok && time.Now().Before(e.expiresAt) {
		p.msMu.Unlock()
		return e.status
	}
	p.msMu.Unlock()

	v, _, _ := p.sf.Do(exchange, func() (any, error) {
		// Double-check inside the singleflight fence in case another goroutine just populated the cache.
		p.msMu.Lock()
		if e, ok := p.msCache[exchange]; ok && time.Now().Before(e.expiresAt) {
			p.msMu.Unlock()
			return e.status, nil
		}
		p.msMu.Unlock()

		status, err := p.fetchMarketStatus(ctx, exchange)
		if err != nil {
			return quote.MarketStatusClosed, nil
		}

		p.msMu.Lock()
		p.msCache[exchange] = marketStatusEntry{
			status:    status,
			expiresAt: time.Now().Add(60 * time.Second),
		}
		p.msMu.Unlock()

		return status, nil
	})

	if s, ok := v.(string); ok {
		return s
	}
	return quote.MarketStatusClosed
}

func (p *FinnhubProvider) fetchMarketStatus(ctx context.Context, exchange string) (string, error) {
	params := url.Values{"exchange": {exchange}, "token": {p.apiKey}}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/stock/market-status?"+params.Encode(), nil)
	if err != nil {
		return "", ErrProviderUnavailable
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return "", ErrProviderUnavailable
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", ErrProviderUnavailable
	}

	var r finnhubMarketStatusResp
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&r); err != nil {
		return "", ErrProviderUnavailable
	}

	if !r.IsOpen {
		return quote.MarketStatusClosed, nil
	}
	switch r.Session {
	case "pre_market":
		return quote.MarketStatusPreMarket, nil
	case "after_market", "post_market":
		return quote.MarketStatusAfterHours, nil
	default:
		return quote.MarketStatusOpen, nil
	}
}

// exchangeShortName maps MIC codes to common exchange abbreviations.
// Falls back to the raw exchange name if MIC is unknown.
func exchangeShortName(mic, fallback string) string {
	switch mic {
	case "XNAS", "XNMS":
		return "NASDAQ"
	case "XNYS":
		return "NYSE"
	case "XASE":
		return "AMEX"
	case "XCBF":
		return "CBOE"
	case "BATS", "EDGX":
		return "BATS"
	case "ARCX":
		return "ARCA"
	}
	if fallback != "" {
		return fallback
	}
	return mic
}

func (p *FinnhubProvider) fetchQuote(ctx context.Context, symbol string) (*finnhubQuoteResp, error) {
	params := url.Values{"symbol": {symbol}, "token": {p.apiKey}}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/quote?"+params.Encode(), nil)
	if err != nil {
		return nil, ErrProviderUnavailable
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, ErrProviderUnavailable
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
		log.Printf("finnhub /quote status=%d body=%q apiKeyLen=%d", resp.StatusCode, body, len(p.apiKey))
		return nil, fmt.Errorf("finnhub /quote status %d: %w", resp.StatusCode, ErrProviderUnavailable)
	}

	var r finnhubQuoteResp
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&r); err != nil {
		return nil, ErrProviderUnavailable
	}
	return &r, nil
}

func (p *FinnhubProvider) fetchMetric(ctx context.Context, symbol string) (*finnhubMetricResp, error) {
	params := url.Values{"symbol": {symbol}, "metric": {"all"}, "token": {p.apiKey}}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/stock/metric?"+params.Encode(), nil)
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

	var r finnhubMetricResp
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&r); err != nil {
		return nil, ErrProviderUnavailable
	}
	return &r, nil
}

type finnhubCalendarResp struct {
	EconomicCalendar []struct {
		Actual   *float64 `json:"actual"`
		Country  string   `json:"country"`
		Estimate *float64 `json:"estimate"`
		Event    string   `json:"event"`
		Impact   string   `json:"impact"`
		Prev     *float64 `json:"prev"`
		Time     string   `json:"time"` // "2024-01-12 13:30:00"
		Unit     string   `json:"unit"`
	} `json:"economicCalendar"`
}

// FetchEconomicCalendar returns upcoming US economic events for the next 7 days.
// High and medium impact events are returned, limited to 6.
func (p *FinnhubProvider) FetchEconomicCalendar(ctx context.Context) ([]quote.EconEvent, error) {
	loc, _ := time.LoadLocation("America/New_York")
	if loc == nil {
		loc = time.UTC
	}
	now := time.Now().In(loc)
	from := now.Format("2006-01-02")
	to := now.AddDate(0, 0, 7).Format("2006-01-02")

	params := url.Values{"from": {from}, "to": {to}, "token": {p.apiKey}}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/calendar/economic?"+params.Encode(), nil)
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

	var r finnhubCalendarResp
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&r); err != nil {
		return nil, ErrProviderUnavailable
	}

	events := make([]quote.EconEvent, 0, 6)
	for _, e := range r.EconomicCalendar {
		if e.Country != "US" {
			continue
		}
		if e.Impact != "high" && e.Impact != "medium" {
			continue
		}
		t, err := time.ParseInLocation("2006-01-02 15:04:05", e.Time, time.UTC)
		when := e.Time
		if err == nil {
			t = t.In(loc)
			when = t.Format("Mon 15:04") + " ET"
		}
		events = append(events, quote.EconEvent{
			Name:    e.Event,
			Country: e.Country,
			When:    when,
			Impact:  e.Impact,
			Actual:  e.Actual,
			Est:     e.Estimate,
			Prev:    e.Prev,
			Unit:    e.Unit,
		})
		if len(events) >= 6 {
			break
		}
	}
	return events, nil
}

func (p *FinnhubProvider) fetchProfile(ctx context.Context, symbol string) (*finnhubProfileResp, error) {
	params := url.Values{"symbol": {symbol}, "token": {p.apiKey}}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/stock/profile2?"+params.Encode(), nil)
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
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&r); err != nil {
		return nil, ErrProviderUnavailable
	}
	return &r, nil
}
