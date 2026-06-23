package render

import (
	"fmt"

	"github.com/dmitriy/curlstreet/internal/quote"
)

type RenderFunc func(quotes []quote.QuoteResult) (string, error)

var rendererMap = map[quote.ResponseFormat]RenderFunc{
	quote.ResponseFormatText: RenderText,
	quote.ResponseFormatHTML: RenderHTML,
	quote.ResponseFormatJSON: RenderJSON,
}

func Render(format quote.ResponseFormat, quotes []quote.QuoteResult) (string, error) {
	fn, ok := rendererMap[format]
	if !ok {
		return "", fmt.Errorf("unknown format: %q", format)
	}
	return fn(quotes)
}
