package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/DmitriyRogo/curlstreet.sh/internal/geo"
)

type stubLocator struct {
	loc geo.Location
	ok  bool
}

func (s stubLocator) Lookup(string) (geo.Location, bool) { return s.loc, s.ok }

func TestRequestLogger_AttachesGeoFieldsOnQuotePath(t *testing.T) {
	// Fly-Client-IP is only trusted when running on fly.io (see clientip.go);
	// without this, clientIP falls back to RemoteAddr and the test fails
	// regardless of environment. Same pattern as clientip_test.go.
	t.Setenv("FLY_APP_NAME", "curlstreet-sh")

	var buf bytes.Buffer
	logger := logrus.New()
	logger.SetOutput(&buf)
	logger.SetFormatter(&logrus.JSONFormatter{})

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })

	srv := &Server{logger: logger, locator: stubLocator{loc: geo.Location{Country: "US", Continent: "NA", City: "Chicago"}, ok: true}}
	srv.handler = srv.requestLogger(mux)

	req := httptest.NewRequest(http.MethodGet, "/AAPL", nil)
	req.Header.Set("Fly-Client-IP", "203.0.113.9")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	require.Contains(t, buf.String(), `"country":"US"`)
	require.Contains(t, buf.String(), `"city":"Chicago"`)
	assert.NotContains(t, buf.String(), "203.0.113.9", "raw client IP must not appear in logs")
	assert.Contains(t, buf.String(), `"ip":"203.0.113.0/24"`)
}

func TestRequestLogger_SkipsGeoOnHealthPath(t *testing.T) {
	var buf bytes.Buffer
	logger := logrus.New()
	logger.SetOutput(&buf)
	logger.SetFormatter(&logrus.JSONFormatter{})

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })

	srv := &Server{logger: logger, locator: stubLocator{loc: geo.Location{Country: "US"}, ok: true}}
	srv.handler = srv.requestLogger(mux)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("Fly-Client-IP", "203.0.113.9")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	assert.NotContains(t, buf.String(), `"country"`)
}
