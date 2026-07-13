package geo

import (
	"net/netip"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/oschwald/maxminddb-golang/v2"
)

// mmdbRecord mirrors the subset of the DB-IP / GeoIP2 City schema this
// package reads. DB-IP's Lite databases use the same field layout as
// MaxMind's GeoIP2 City, so this decodes both interchangeably.
type mmdbRecord struct {
	Country struct {
		IsoCode string `maxminddb:"iso_code"`
	} `maxminddb:"country"`
	Continent struct {
		Code string `maxminddb:"code"`
	} `maxminddb:"continent"`
	City struct {
		Names map[string]string `maxminddb:"names"`
	} `maxminddb:"city"`
	Subdivisions []struct {
		IsoCode string `maxminddb:"iso_code"`
	} `maxminddb:"subdivisions"`
	Location struct {
		Latitude  float64 `maxminddb:"latitude"`
		Longitude float64 `maxminddb:"longitude"`
	} `maxminddb:"location"`
}

type cacheEntry struct {
	loc Location
	ok  bool
}

// defaultCacheSize bounds the IP→Location LRU. Sized generously above typical
// concurrent-client counts; eviction just costs a repeat client one more mmdb
// lookup, never a correctness issue.
const defaultCacheSize = 4096

// MMDBLocator resolves IPs using a memory-mapped DB-IP/GeoIP2 City database.
type MMDBLocator struct {
	db    *maxminddb.Reader
	cache *lru.Cache[string, cacheEntry]
}

// NewMMDBLocator opens the MMDB file at path via mmap. The returned locator
// must be closed with Close when no longer needed.
func NewMMDBLocator(path string) (*MMDBLocator, error) {
	db, err := maxminddb.Open(path)
	if err != nil {
		return nil, err
	}
	cache, err := lru.New[string, cacheEntry](defaultCacheSize)
	if err != nil {
		db.Close()
		return nil, err
	}
	return &MMDBLocator{db: db, cache: cache}, nil
}

// Lookup resolves ip to a Location, consulting the LRU cache first.
func (l *MMDBLocator) Lookup(ip string) (Location, bool) {
	if entry, ok := l.cache.Get(ip); ok {
		return entry.loc, entry.ok
	}
	loc, ok := l.lookupUncached(ip)
	l.cache.Add(ip, cacheEntry{loc: loc, ok: ok})
	return loc, ok
}

func (l *MMDBLocator) lookupUncached(ip string) (Location, bool) {
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return Location{}, false
	}

	result := l.db.Lookup(addr)
	if result.Err() != nil || !result.Found() {
		return Location{}, false
	}

	var rec mmdbRecord
	if err := result.Decode(&rec); err != nil {
		return Location{}, false
	}

	region := ""
	if len(rec.Subdivisions) > 0 {
		region = rec.Subdivisions[0].IsoCode
	}

	return Location{
		Country:   rec.Country.IsoCode,
		Continent: rec.Continent.Code,
		City:      rec.City.Names["en"],
		Region:    region,
		Lat:       rec.Location.Latitude,
		Lon:       rec.Location.Longitude,
	}, true
}

// Close releases the underlying memory-mapped database.
func (l *MMDBLocator) Close() error {
	return l.db.Close()
}
