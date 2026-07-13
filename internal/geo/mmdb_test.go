package geo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMMDBLocator_Lookup_KnownIP(t *testing.T) {
	loc, err := NewMMDBLocator("testdata/GeoIP2-City-Test.mmdb")
	require.NoError(t, err)
	defer loc.Close()

	got, ok := loc.Lookup("81.2.69.142")
	require.True(t, ok)
	assert.Equal(t, "GB", got.Country)
	assert.Equal(t, "EU", got.Continent)
	assert.Equal(t, "London", got.City)
	assert.Equal(t, "ENG", got.Region)
	assert.InDelta(t, 51.5142, got.Lat, 0.001)
	assert.InDelta(t, -0.0931, got.Lon, 0.001)
}

func TestMMDBLocator_Lookup_NotFound(t *testing.T) {
	loc, err := NewMMDBLocator("testdata/GeoIP2-City-Test.mmdb")
	require.NoError(t, err)
	defer loc.Close()

	for _, ip := range []string{"10.0.0.1", "192.168.1.1", "127.0.0.1"} {
		_, ok := loc.Lookup(ip)
		assert.Falsef(t, ok, "expected %s to be absent from the test fixture", ip)
	}
}

func TestMMDBLocator_Lookup_Garbage(t *testing.T) {
	loc, err := NewMMDBLocator("testdata/GeoIP2-City-Test.mmdb")
	require.NoError(t, err)
	defer loc.Close()

	_, ok := loc.Lookup("not-an-ip")
	assert.False(t, ok)
}

func TestMMDBLocator_Lookup_ServedFromCache(t *testing.T) {
	loc, err := NewMMDBLocator("testdata/GeoIP2-City-Test.mmdb")
	require.NoError(t, err)
	defer loc.Close()

	first, ok := loc.Lookup("81.2.69.142")
	require.True(t, ok)
	second, ok := loc.Lookup("81.2.69.142")
	require.True(t, ok)
	assert.Equal(t, first, second)
}

func TestNewMMDBLocator_MissingFile(t *testing.T) {
	_, err := NewMMDBLocator("testdata/does-not-exist.mmdb")
	assert.Error(t, err)
}
