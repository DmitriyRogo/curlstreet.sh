package quote

import (
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// P1: NormaliseSymbol is case-insensitive — output always matches upper-cased input
func TestNormaliseSymbolCaseInsensitive(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		s := rapid.StringMatching(`[A-Za-z0-9\-\.]{1,10}`).Draw(t, "symbol")
		if NormaliseSymbol(s) != NormaliseSymbol(strings.ToUpper(s)) {
			t.Fatalf("NormaliseSymbol(%q) != NormaliseSymbol(%q)", s, strings.ToUpper(s))
		}
	})
}
