package provider

import (
	"context"

	"github.com/DmitriyRogo/curlstreet.sh/internal/quote"
)

type DataProvider interface {
	Fetch(ctx context.Context, symbol string) (*quote.Quote, error)
}
