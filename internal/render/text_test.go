package render

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"pgregory.net/rapid"
	"github.com/dmitriy/curlstreet/internal/quote"
)

func makeTestStockQuote(price, change, pct float64) *quote.Quote {
	vol := int64(1000000)
	avg := int64(900000)
	h, l := price * 1.2, price * 0.8
	status := quote.MarketStatusOpen
	return &quote.Quote{
		Symbol: "AAPL", Name: "Apple Inc.", Price: price,
		Change: change, ChangePercent: pct,
		Volume: &vol, AvgVolume: &avg,
		High52W: &h, Low52W: &l,
		Currency: "USD", MarketStatus: &status,
		AssetType: quote.AssetTypeStock, UpdatedAt: time.Now(),
	}
}

func makeTestCryptoQuote(price, change, pct float64) *quote.Quote {
	vol := int64(28000000000)
	h, l := price * 1.05, price * 0.95
	return &quote.Quote{
		Symbol: "BTC", Name: "Bitcoin", Price: price,
		Change: change, ChangePercent: pct,
		Volume: &vol,
		High52W: &h, Low52W: &l,
		Currency: "USD", MarketStatus: nil,
		AssetType: quote.AssetTypeCrypto, UpdatedAt: time.Now(),
	}
}

// P5: Every line in plain-text output is ≤ 80 characters (excluding ANSI escape sequences)
func TestRenderTextLineWidth(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		price := rapid.Float64Range(0.000001, 100000).Draw(t, "price")
		change := rapid.Float64Range(-1000, 1000).Draw(t, "change")
		pct := rapid.Float64Range(-50, 50).Draw(t, "pct")

		q := makeTestStockQuote(price, change, pct)
		out, err := RenderText([]quote.QuoteResult{{Quote: q}})
		assert.NoError(t, err)

		for _, line := range strings.Split(out, "\n") {
			// Strip ANSI escape codes for length check
			plain := stripANSI(line)
			assert.LessOrEqual(t, len(plain), 80, "line too long: %q", plain)
		}
	})
}

// P6: Symbol, price, change present; green on +, red on −
func TestRenderTextRequiredFields(t *testing.T) {
	q := makeTestStockQuote(189.45, 1.23, 0.65)
	out, err := RenderText([]quote.QuoteResult{{Quote: q}})
	assert.NoError(t, err)
	assert.Contains(t, out, "AAPL")
	assert.Contains(t, out, "189.45")
	assert.Contains(t, out, "+1.23")
}

func TestRenderTextGreenOnPositive(t *testing.T) {
	q := makeTestStockQuote(100, 5.0, 5.0)
	out, err := RenderText([]quote.QuoteResult{{Quote: q}})
	assert.NoError(t, err)
	// ANSI green escape should be present
	assert.Contains(t, out, "\x1b[")
	assert.Contains(t, out, "+5.00")
}

func TestRenderTextRedOnNegative(t *testing.T) {
	q := makeTestStockQuote(100, -5.0, -5.0)
	out, err := RenderText([]quote.QuoteResult{{Quote: q}})
	assert.NoError(t, err)
	assert.Contains(t, out, "\x1b[")
	assert.Contains(t, out, "-5.00")
}

// P11: Crypto omits market status, uses 24h labels
func TestCryptoOmitsMarketStatus(t *testing.T) {
	q := makeTestCryptoQuote(62000.0, 1200.0, 1.97)
	out, err := RenderText([]quote.QuoteResult{{Quote: q}})
	assert.NoError(t, err)
	assert.NotContains(t, out, "LIVE")
	assert.NotContains(t, out, "LAST CLOSE")
	assert.Contains(t, out, "24h High")
	assert.Contains(t, out, "24h Low")
}

// P12: Crypto decimal precision — ≥$0.01 → 2dp; <$0.01 → up to 8dp
func TestCryptoDecimalPrecision(t *testing.T) {
	t.Run("above_cent", func(t *testing.T) {
		q := makeTestCryptoQuote(62000.0, 0, 0)
		out, _ := RenderText([]quote.QuoteResult{{Quote: q}})
		assert.Contains(t, out, "62000.00")
	})
	t.Run("below_cent", func(t *testing.T) {
		q := makeTestCryptoQuote(0.000001, 0, 0)
		out, _ := RenderText([]quote.QuoteResult{{Quote: q}})
		assert.Contains(t, out, "0.00000100")
	})
}

func TestRenderTextDivider(t *testing.T) {
	q1 := makeTestStockQuote(100, 1, 1)
	q2 := makeTestStockQuote(200, -1, -1)
	q1.Symbol = "AAPL"
	q2.Symbol = "TSLA"
	out, err := RenderText([]quote.QuoteResult{{Quote: q1}, {Quote: q2}})
	assert.NoError(t, err)
	assert.Contains(t, out, "──")
}

// stripANSI removes ANSI escape codes from s for length checking.
func stripANSI(s string) string {
	var result strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			// skip until 'm'
			i += 2
			for i < len(s) && s[i] != 'm' {
				i++
			}
			i++ // skip 'm'
		} else {
			result.WriteByte(s[i])
			i++
		}
	}
	return result.String()
}
