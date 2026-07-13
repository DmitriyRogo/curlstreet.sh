package geo

// Location is the resolved geolocation of an IP address.
type Location struct {
	Country   string // ISO 3166-1 alpha-2 code, e.g. "US"
	Continent string // ISO continent code, e.g. "NA"
	City      string // English city name, e.g. "Chicago"
	Region    string // First-level subdivision ISO code, e.g. "IL"
	Lat       float64
	Lon       float64
}

// Locator resolves an IP address to a Location. ok is false when the address
// is private, reserved, unparseable, or absent from the underlying database.
type Locator interface {
	Lookup(ip string) (loc Location, ok bool)
}
