package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMaskIP_IPv4TruncatesToSlash24(t *testing.T) {
	assert.Equal(t, "203.0.113.0/24", maskIP("203.0.113.42"))
}

func TestMaskIP_IPv6TruncatesToSlash48(t *testing.T) {
	// 48 bits = the first 3 hextets (2001:0db8:1234); the rest is zeroed.
	// Verified against Go's net.IP.Mask(net.CIDRMask(48, 128)) directly —
	// the brief's original expected literal ("2001:db8::/48") corresponds to
	// a /32 mask, not /48, and was a bug in the brief.
	assert.Equal(t, "2001:db8:1234::/48", maskIP("2001:db8:1234:5678::1"))
}

func TestMaskIP_Garbage(t *testing.T) {
	assert.Equal(t, "unknown", maskIP("not-an-ip"))
}
