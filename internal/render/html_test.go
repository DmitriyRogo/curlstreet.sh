package render

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/DmitriyRogo/curlstreet.sh/internal/quote"
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

func TestAnsiToHTML_RejectsJavascriptScheme(t *testing.T) {
	// OSC 8 hyperlink with javascript: scheme must not produce an <a> tag
	malicious := "\033]8;;javascript:alert(1)\033\\" + "click me" + "\033]8;;\033\\"
	got := ansiToHTML(malicious)
	assert.NotContains(t, got, "<a ", "javascript: URL must not produce an anchor tag")
	assert.Contains(t, got, "click me", "link text must still be rendered")
}

func TestAnsiToHTML_AllowsHttpsScheme(t *testing.T) {
	link := "\033]8;;https://example.com\033\\" + "example" + "\033]8;;\033\\"
	got := ansiToHTML(link)
	assert.Contains(t, got, `href="https://example.com"`)
}

func TestRenderError_JSONEscaping(t *testing.T) {
	msg := `symbol "BTC/USD" not found: path\value`
	got := RenderError(400, msg, quote.ResponseFormatJSON)
	var parsed map[string]string
	require.NoError(t, json.Unmarshal([]byte(got), &parsed), "must be valid JSON: %s", got)
	assert.Equal(t, msg, parsed["error"])
}
