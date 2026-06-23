package provider

import "errors"

var ErrSymbolNotFound = errors.New("symbol not found")
var ErrProviderUnavailable = errors.New("provider unavailable")
