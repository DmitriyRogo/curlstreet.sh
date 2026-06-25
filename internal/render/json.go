package render

import (
	"encoding/json"

	"github.com/DmitriyRogo/curlstreet.sh/internal/quote"
)

func RenderJSON(quotes []quote.QuoteResult) (string, error) {
	var v any
	if len(quotes) == 1 {
		v = quotes[0]
	} else {
		v = quotes
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
