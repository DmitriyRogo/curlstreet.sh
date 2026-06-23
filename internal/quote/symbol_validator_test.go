package quote

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"pgregory.net/rapid"
)

// P2: ValidateSymbol accepts valid symbols and rejects invalid ones
func TestValidateSymbolAcceptsValid(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		s := rapid.StringMatching(`[A-Za-z0-9\-\.]{1,10}`).Draw(t, "symbol")
		assert.NoError(t, ValidateSymbol(s))
	})
}

func TestValidateSymbolRejectsEmpty(t *testing.T) {
	assert.Error(t, ValidateSymbol(""))
}

func TestValidateSymbolRejectsTooLong(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// 11+ chars of valid chars
		s := rapid.StringMatching(`[A-Za-z0-9]{11,20}`).Draw(t, "long")
		assert.Error(t, ValidateSymbol(s))
	})
}

func TestValidateSymbolRejectsBadChars(t *testing.T) {
	bad := []string{"AA BB", "AA@BB", "AA#1", "$$", "aaaa!"}
	for _, s := range bad {
		assert.Error(t, ValidateSymbol(s), "expected error for %q", s)
	}
}
