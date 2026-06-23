package quote

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"pgregory.net/rapid"
)

// P3: Explicit format param overrides User-Agent for any combination
func TestExplicitFormatParamOverridesUA(t *testing.T) {
	params := []string{"text", "html", "json"}
	rapid.Check(t, func(t *rapid.T) {
		ua := rapid.String().Draw(t, "ua")
		param := rapid.SampledFrom(params).Draw(t, "param")
		got := DetectFormat(ua, param)
		assert.Equal(t, ResponseFormat(param), got)
	})
}

// P4: UA heuristic when no explicit param
func TestUAHeuristicNoParam(t *testing.T) {
	browserUAs := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
		"Chrome/112.0 Safari/537.36",
		"Firefox/89.0",
		"Opera/9.80",
	}
	for _, ua := range browserUAs {
		assert.Equal(t, ResponseFormatHTML, DetectFormat(ua, ""), "expected HTML for browser UA: %q", ua)
	}

	rapid.Check(t, func(t *rapid.T) {
		// Generate UA that contains none of the browser keywords
		ua := rapid.StringMatching(`[a-z0-9/\.\-\_]{0,40}`).Draw(t, "non-browser-ua")
		got := DetectFormat(ua, "")
		assert.Equal(t, ResponseFormatText, got)
	})
}

func TestDetectFormatInvalidParamFallsBackToUA(t *testing.T) {
	// Unknown format param → treat as no param, use UA heuristic
	assert.Equal(t, ResponseFormatText, DetectFormat("curl/7.88.0", "xml"))
	assert.Equal(t, ResponseFormatHTML, DetectFormat("Mozilla/5.0", "xml"))
}
