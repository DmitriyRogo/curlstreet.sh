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
	Currency      string    `json:"currency"`
	MarketStatus  *string   `json:"market_status"`
	AssetType     AssetType `json:"asset_type"`
	UpdatedAt     time.Time `json:"updated_at"`
	CacheHit      bool      `json:"-"`
}

type SymbolError struct {
	Symbol  string `json:"symbol"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type QuoteResult struct {
	Quote *Quote       `json:"quote,omitempty"`
	Err   *SymbolError `json:"error,omitempty"`
}
