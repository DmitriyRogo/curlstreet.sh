package quote

import (
	"fmt"
	"regexp"
	"strings"
)

var symbolPattern = regexp.MustCompile(`^[A-Za-z0-9\-\.]{1,10}$`)

func ValidateSymbol(s string) error {
	if !symbolPattern.MatchString(s) {
		return fmt.Errorf("invalid symbol '%s'. Symbols must be 1–10 alphanumeric characters (hyphens and dots allowed)", s)
	}
	return nil
}

func NormaliseSymbol(s string) string {
	return strings.ToUpper(s)
}
