package server

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClientIP_FlyClientIPWinsOverEverything(t *testing.T) {
	t.Setenv("FLY_APP_NAME", "curlstreet-sh")

	r := httptest.NewRequest(http.MethodGet, "/AAPL", nil)
	r.RemoteAddr = "10.0.0.5:1234"
	r.Header.Set("Fly-Client-IP", "203.0.113.9")
	r.Header.Set("X-Forwarded-For", "198.51.100.1")

	_, trustedNet := mustParseCIDR(t, "10.0.0.0/8")
	assert.Equal(t, "203.0.113.9", clientIP(r, trustedNet))
}

func TestClientIP_FlyClientIPIgnoredWhenNotOnFly(t *testing.T) {
	t.Setenv("FLY_APP_NAME", "")

	r := httptest.NewRequest(http.MethodGet, "/AAPL", nil)
	r.RemoteAddr = "203.0.113.5:1234"
	r.Header.Set("Fly-Client-IP", "9.9.9.9") // attacker-supplied, must be ignored off-Fly

	assert.Equal(t, "203.0.113.5", clientIP(r, nil))
}

func TestClientIP_TrustedProxyXFFFallback(t *testing.T) {
	t.Setenv("FLY_APP_NAME", "")

	r := httptest.NewRequest(http.MethodGet, "/AAPL", nil)
	r.RemoteAddr = "10.0.0.5:1234" // inside the trusted CIDR
	r.Header.Set("X-Forwarded-For", "198.51.100.1, 10.0.0.5")

	_, trustedNet := mustParseCIDR(t, "10.0.0.0/8")
	assert.Equal(t, "198.51.100.1", clientIP(r, trustedNet))
}

func TestClientIP_UntrustedRemoteIgnoresXFF(t *testing.T) {
	t.Setenv("FLY_APP_NAME", "")

	r := httptest.NewRequest(http.MethodGet, "/AAPL", nil)
	r.RemoteAddr = "203.0.113.5:1234" // NOT inside the trusted CIDR
	r.Header.Set("X-Forwarded-For", "198.51.100.1")

	_, trustedNet := mustParseCIDR(t, "10.0.0.0/8")
	assert.Equal(t, "203.0.113.5", clientIP(r, trustedNet))
}

func TestClientIP_NoTrustedProxyFallsBackToRemoteAddr(t *testing.T) {
	t.Setenv("FLY_APP_NAME", "")

	r := httptest.NewRequest(http.MethodGet, "/AAPL", nil)
	r.RemoteAddr = "203.0.113.5:1234"

	assert.Equal(t, "203.0.113.5", clientIP(r, nil))
}

func TestParseTrustedProxy_EmptyAndInvalid(t *testing.T) {
	assert.Nil(t, parseTrustedProxy(""))
	assert.Nil(t, parseTrustedProxy("not-a-cidr"))
}

func mustParseCIDR(t *testing.T, cidr string) (string, *net.IPNet) {
	t.Helper()
	net := parseTrustedProxy(cidr)
	if net == nil {
		t.Fatalf("expected %q to parse", cidr)
	}
	return cidr, net
}
