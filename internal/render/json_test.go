package render

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
	"github.com/dmitriy/curlstreet/internal/quote"
)

// P7: JSON completeness — all required keys present; nil → null; updated_at is RFC 3339
func TestRenderJSONCompleteness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		price := rapid.Float64Range(0.01, 100000).Draw(t, "price")
		now := time.Now().UTC()

		q := &quote.Quote{
			Symbol: "AAPL", Name: "Apple Inc.", Price: price,
			Change: 1.0, ChangePercent: 0.5,
			Volume: nil, AvgVolume: nil, High52W: nil, Low52W: nil,
			Currency: "USD", MarketStatus: nil,
			AssetType: quote.AssetTypeStock, UpdatedAt: now,
		}
		out, err := RenderJSON([]quote.QuoteResult{{Quote: q}})
		require.NoError(t, err)

		// Single quote → unwrapped object
		var obj map[string]any
		require.NoError(t, json.Unmarshal([]byte(out), &obj))

		// required keys
		for _, key := range []string{"symbol", "name", "price", "change", "change_percent", "currency", "asset_type", "updated_at"} {
			_, ok := obj["quote"].(map[string]any)[key]
			assert.True(t, ok, "missing key: %s", key)
		}

		qObj := obj["quote"].(map[string]any)

		// nil pointer fields must serialize as null
		assert.Nil(t, qObj["volume"])
		assert.Nil(t, qObj["avg_volume"])
		assert.Nil(t, qObj["high_52w"])
		assert.Nil(t, qObj["low_52w"])
		assert.Nil(t, qObj["market_status"])

		// updated_at is RFC 3339
		updatedAt, ok := qObj["updated_at"].(string)
		assert.True(t, ok)
		_, err = time.Parse(time.RFC3339, updatedAt)
		assert.NoError(t, err, "updated_at not RFC3339: %s", updatedAt)

		// CacheHit must not appear
		_, hasCacheHit := qObj["cache_hit"]
		assert.False(t, hasCacheHit, "CacheHit must not appear in JSON")
	})
}

func TestRenderJSONBatchIsArray(t *testing.T) {
	vol := int64(100)
	q1 := &quote.Quote{Symbol: "AAPL", Name: "Apple", Price: 100, Currency: "USD", AssetType: quote.AssetTypeStock, UpdatedAt: time.Now(), Volume: &vol}
	q2 := &quote.Quote{Symbol: "TSLA", Name: "Tesla", Price: 200, Currency: "USD", AssetType: quote.AssetTypeStock, UpdatedAt: time.Now(), Volume: &vol}

	out, err := RenderJSON([]quote.QuoteResult{{Quote: q1}, {Quote: q2}})
	require.NoError(t, err)

	var arr []any
	require.NoError(t, json.Unmarshal([]byte(out), &arr))
	assert.Len(t, arr, 2)
}
