package server

import "net"

// maskIP truncates ip to a /24 (IPv4) or /48 (IPv6) network for privacy-safe
// logging — coarse enough to avoid identifying an individual alongside the
// city-level geo fields, precise enough to still be useful for debugging.
func maskIP(ip string) string {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return "unknown"
	}
	if v4 := parsed.To4(); v4 != nil {
		return net.IPv4(v4[0], v4[1], v4[2], 0).String() + "/24"
	}
	return parsed.Mask(net.CIDRMask(48, 128)).String() + "/48"
}
