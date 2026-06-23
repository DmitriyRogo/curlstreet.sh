package provider

import (
	"context"

	"github.com/dmitriy/curlstreet/internal/quote"
)

type DataProvider interface {
	Fetch(ctx context.Context, symbol string) (*quote.Quote, error)
}
