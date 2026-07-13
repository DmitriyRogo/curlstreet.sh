package geo

// NopLocator resolves nothing. It's used when no geolocation database is
// configured, so the server still boots with location tracking disabled.
type NopLocator struct{}

// Lookup always reports a miss.
func (NopLocator) Lookup(string) (Location, bool) { return Location{}, false }
