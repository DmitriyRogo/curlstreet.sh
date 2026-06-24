package quote

import "time"

type AssetType string

const (
	AssetTypeStock  AssetType = "STOCK"
	AssetTypeCrypto AssetType = "CRYPTO"
)

const (
	MarketStatusOpen       = "OPEN"
	MarketStatusClosed     = "CLOSED"
	MarketStatusPreMarket  = "PRE_MARKET"
	MarketStatusAfterHours = "AFTER_HOURS"
)

type ResponseFormat string

const (
	ResponseFormatText ResponseFormat = "text"
	ResponseFormatHTML ResponseFormat = "html"
	ResponseFormatJSON ResponseFormat = "json"
)

type Quote struct {
	Symbol        string    `json:"symbol"`
	Name          string    `json:"name"`
	Price         float64   `json:"price"`
	Change        float64   `json:"change"`
	ChangePercent float64   `json:"change_percent"`
	Volume        *int64    `json:"volume"`
	AvgVolume     *int64    `json:"avg_volume"`
	High52W       *float64  `json:"high_52w"`
	Low52W        *float64  `json:"low_52w"`
	MarketCap     *int64    `json:"market_cap"`
	Currency      string    `json:"currency"`
	MarketStatus  *string   `json:"market_status"`
	AssetType     AssetType `json:"asset_type"`
	Exchange      string    `json:"exchange"`
	ExchangeCode  string    `json:"exchange_code,omitempty"`
	SecurityType  string    `json:"security_type,omitempty"`
	Sector        string    `json:"sector"`
	Industry      string    `json:"industry"`
	UpdatedAt     time.Time `json:"updated_at"`
	CacheHit      bool      `json:"-"`
}

// EconEvent represents a single entry from an economic data calendar.
type EconEvent struct {
	Name    string   `json:"event"`
	Country string   `json:"country"`
	When    string   `json:"time"`    // formatted display time, e.g. "Thu 08:30 ET"
	Impact  string   `json:"impact"`  // "low", "medium", "high"
	Actual  *float64 `json:"actual"`
	Est     *float64 `json:"estimate"`
	Prev    *float64 `json:"prev"`
	Unit    string   `json:"unit"`
}

type SymbolError struct {
	Symbol  string `json:"symbol"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type QuoteResult struct {
	Quote        *Quote       `json:"quote,omitempty"`
	Err          *SymbolError `json:"error,omitempty"`
	IsMarket     bool         `json:"-"`
	IsPulse      bool         `json:"-"`
	DisplayLabel string       `json:"-"`
}
