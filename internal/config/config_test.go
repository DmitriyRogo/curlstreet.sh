package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_GeoDBPathDefault(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "does-not-exist.yaml"))
	require.NoError(t, err)
	assert.Equal(t, "/data/dbip-city-lite.mmdb", cfg.Geo.DBPath)
}

func TestLoad_GeoDBPathOverride(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("geo:\n  db_path: \"/custom/geo.mmdb\"\n"), 0o600))

	cfg, err := Load(path)
	require.NoError(t, err)
	assert.Equal(t, "/custom/geo.mmdb", cfg.Geo.DBPath)
}
