package render

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/dmitriy/curlstreet/internal/quote"
)

func TestRenderHTMLContainsDoctype(t *testing.T) {
	vol := int64(1000)
	h, l := 200.0, 150.0
	status := quote.MarketStatusOpen
	q := &quote.Quote{
		Symbol: "AAPL", Name: "Apple Inc.", Price: 189.45,
		Change: 1.23, ChangePercent: 0.65,
		Volume: &vol, High52W: &h, Low52W: &l,
		Currency: "USD", MarketStatus: &status,
		AssetType: quote.AssetTypeStock, UpdatedAt: time.Now(),
	}
	out, err := RenderHTML([]quote.QuoteResult{{Quote: q}})
	require.NoError(t, err)
	assert.Contains(t, out, "<!DOCTYPE html>")
	assert.Contains(t, out, "<pre")
	assert.Contains(t, out, "AAPL")
}

func TestRenderHTMLHasMonospaceFont(t *testing.T) {
	q := &quote.Quote{
		Symbol: "ETH", Name: "Ethereum", Price: 3000, Currency: "USD",
		AssetType: quote.AssetTypeCrypto, UpdatedAt: time.Now(),
	}
	out, err := RenderHTML([]quote.QuoteResult{{Quote: q}})
	require.NoError(t, err)
	assert.Contains(t, out, "monospace")
}
