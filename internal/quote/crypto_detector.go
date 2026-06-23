package quote

import (
	"github.com/dmitriy/curlstreet/assets"
	"gopkg.in/yaml.v3"
)

var cryptoIDs map[string]string

func init() {
	if err := yaml.Unmarshal(assets.CryptoIDsYAML, &cryptoIDs); err != nil {
		panic("failed to parse crypto_ids.yaml: " + err.Error())
	}
}

// IsCrypto returns true if symbol is a known cryptocurrency.
func IsCrypto(symbol string) bool {
	_, ok := cryptoIDs[symbol]
	return ok
}

// CoinGeckoID returns the CoinGecko ID for a crypto symbol.
func CoinGeckoID(symbol string) (string, bool) {
	id, ok := cryptoIDs[symbol]
	return id, ok
}
