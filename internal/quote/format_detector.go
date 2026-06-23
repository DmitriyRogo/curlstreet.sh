package quote

import "strings"

func DetectFormat(userAgent, formatParam string) ResponseFormat {
	switch formatParam {
	case "text", "html", "json":
		return ResponseFormat(formatParam)
	}

	for _, browser := range []string{"Mozilla", "Chrome", "Safari", "Firefox", "Opera"} {
		if strings.Contains(userAgent, browser) {
			return ResponseFormatHTML
		}
	}
	return ResponseFormatText
}
