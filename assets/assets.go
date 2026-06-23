// Package assets embeds static files for the curlstreet service.
package assets

import _ "embed"

//go:embed crypto_ids.yaml
var CryptoIDsYAML []byte

//go:embed templates/base.html
var BaseHTML []byte
