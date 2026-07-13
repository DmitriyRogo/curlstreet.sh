package server

import (
	"net"
	"net/http"
	"os"
	"strings"
)

// isFlyEnvironment reports whether this process is running as a fly.io app.
// fly.io sets FLY_APP_NAME on every deployed machine; its presence means
// fly-proxy is the sole ingress path (this repo's fly.toml defines no raw
// TCP passthrough), so fly-proxy — never an external client — is the only
// thing that can be setting Fly-Client-IP.
func isFlyEnvironment() bool {
	return os.Getenv("FLY_APP_NAME") != ""
}

// parseTrustedProxy parses cidr into a *net.IPNet, or nil when cidr is empty
// or invalid (invalid input disables trusted-proxy handling rather than
// failing startup).
func parseTrustedProxy(cidr string) *net.IPNet {
	if cidr == "" {
		return nil
	}
	_, trustedNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil
	}
	return trustedNet
}

// clientIP resolves the real client address for a request. Priority:
//  1. Fly-Client-IP — the authoritative client IP fly.io's edge proxy sets on
//     every request, but only trusted when isFlyEnvironment reports we're
//     actually running behind fly-proxy. Outside that environment this
//     header is fully attacker-controlled, and trusting it would let a
//     client spoof its identity to bypass IP-based rate limiting.
//  2. X-Forwarded-For's leftmost entry, but only when RemoteAddr falls inside
//     trustedNet (a reverse proxy we trust to set that header honestly).
//  3. RemoteAddr as a last resort.
func clientIP(r *http.Request, trustedNet *net.IPNet) string {
	if isFlyEnvironment() {
		if fly := r.Header.Get("Fly-Client-IP"); fly != "" {
			return fly
		}
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		ip = r.RemoteAddr
	}

	if trustedNet != nil {
		if remoteIP := net.ParseIP(ip); remoteIP != nil && trustedNet.Contains(remoteIP) {
			if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
				if parts := strings.SplitN(xff, ",", 2); len(parts) > 0 {
					if candidate := strings.TrimSpace(parts[0]); candidate != "" {
						return candidate
					}
				}
			}
		}
	}

	return ip
}
