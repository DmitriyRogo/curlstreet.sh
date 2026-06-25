package render

import (
	"fmt"

	"github.com/DmitriyRogo/curlstreet.sh/internal/quote"
)

func Render(format quote.ResponseFormat, quotes []quote.QuoteResult, events ...quote.EconEvent) (string, error) {
	switch format {
	case quote.ResponseFormatText:
		return RenderText(quotes, events...)
	case quote.ResponseFormatHTML:
		return RenderHTML(quotes, events...)
	case quote.ResponseFormatJSON:
		return RenderJSON(quotes)
	default:
		return "", fmt.Errorf("unknown format: %q", format)
	}
}
