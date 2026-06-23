package render

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"strconv"
	"strings"

	"github.com/dmitriy/curlstreet/assets"
	"github.com/dmitriy/curlstreet/internal/quote"
)

var htmlTemplate = template.Must(template.New("base").Parse(string(assets.BaseHTML)))

type htmlData struct {
	Title string
	Body  template.HTML
}

func RenderHTML(quotes []quote.QuoteResult, events ...quote.EconEvent) (string, error) {
	ansi, err := RenderText(quotes, events...)
	if err != nil {
		return "", err
	}

	rendered := ansiToHTML(ansi)

	title := "curlstreet"
	for _, qr := range quotes {
		if qr.Quote != nil && !qr.IsMarket {
			title = qr.Quote.Symbol + " — " + formatPriceCompact(qr.Quote)
			break
		}
	}

	var buf bytes.Buffer
	if err := htmlTemplate.Execute(&buf, htmlData{
		Title: title,
		Body:  template.HTML(rendered),
	}); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func RenderError(code int, msg string, format quote.ResponseFormat) string {
	switch format {
	case quote.ResponseFormatJSON:
		b, _ := json.Marshal(map[string]string{"error": msg})
		return string(b)
	case quote.ResponseFormatHTML:
		escaped := template.HTMLEscapeString(msg)
		var buf bytes.Buffer
		htmlTemplate.Execute(&buf, htmlData{
			Title: "Error",
			Body:  template.HTML(`<span style="color:#e8675f">` + escaped + `</span>`),
		})
		return buf.String()
	default:
		return msg
	}
}

// ansiToHTML converts 24-bit ANSI SGR color codes and OSC 8 hyperlinks to
// inline HTML. It handles:
//   - Foreground (38;2;R;G;B) and background (48;2;R;G;B) SGR sequences
//   - Reset (0m) to close the current span
//   - OSC 8 hyperlinks (ESC]8;;URL ST → <a href>, ESC]8;;ST → </a>)
//
// All other ANSI sequences are stripped; text is HTML-escaped.
func ansiToHTML(input string) string {
	var result strings.Builder
	spanOpen := false
	linkOpen := false
	i := 0

	closeSpan := func() {
		if spanOpen {
			result.WriteString("</span>")
			spanOpen = false
		}
	}
	closeLink := func() {
		if linkOpen {
			result.WriteString("</a>")
			linkOpen = false
		}
	}

	for i < len(input) {
		if input[i] != 0x1b || i+1 >= len(input) {
			switch input[i] {
			case '<':
				result.WriteString("&lt;")
			case '>':
				result.WriteString("&gt;")
			case '&':
				result.WriteString("&amp;")
			default:
				result.WriteByte(input[i])
			}
			i++
			continue
		}

		switch input[i+1] {
		case '[':
			// SGR sequence: scan to final 'm'
			j := i + 2
			for j < len(input) && (input[j] < 0x40 || input[j] > 0x7e) {
				j++
			}
			if j >= len(input) {
				i++
				continue
			}
			if input[j] != 'm' {
				i = j + 1
				continue
			}

			params := strings.Split(input[i+2:j], ";")
			fgR, fgG, fgB := -1, -1, -1
			bgR, bgG, bgB := -1, -1, -1
			reset := false

			for k := 0; k < len(params); k++ {
				n, _ := strconv.Atoi(params[k])
				switch n {
				case 0:
					reset = true
				case 38:
					if k+4 < len(params) && params[k+1] == "2" {
						fgR, _ = strconv.Atoi(params[k+2])
						fgG, _ = strconv.Atoi(params[k+3])
						fgB, _ = strconv.Atoi(params[k+4])
						k += 4
					}
				case 48:
					if k+4 < len(params) && params[k+1] == "2" {
						bgR, _ = strconv.Atoi(params[k+2])
						bgG, _ = strconv.Atoi(params[k+3])
						bgB, _ = strconv.Atoi(params[k+4])
						k += 4
					}
				}
			}

			if reset {
				closeSpan()
			} else if fgR >= 0 || bgR >= 0 {
				closeSpan()
				var style strings.Builder
				if fgR >= 0 {
					fmt.Fprintf(&style, "color:#%02x%02x%02x;", fgR, fgG, fgB)
				}
				if bgR >= 0 {
					fmt.Fprintf(&style, "background-color:#%02x%02x%02x;", bgR, bgG, bgB)
				}
				result.WriteString(`<span style="` + strings.TrimRight(style.String(), ";") + `">`)
				spanOpen = true
			}
			i = j + 1

		case ']':
			// OSC sequence: scan to BEL (0x07) or ST (ESC \)
			j := i + 2
			for j < len(input) {
				if input[j] == 0x07 {
					break
				}
				if input[j] == 0x1b && j+1 < len(input) && input[j+1] == '\\' {
					break
				}
				j++
			}
			oscContent := input[i+2 : j]
			// Advance past terminator
			if j < len(input) {
				if input[j] == 0x07 {
					i = j + 1
				} else { // ESC '\'
					i = j + 2
				}
			} else {
				i = j
			}

			// Handle OSC 8 hyperlinks: "8;;URL" to open, "8;;" to close
			if strings.HasPrefix(oscContent, "8;;") {
				rawURL := oscContent[3:]
				if rawURL == "" {
					closeLink()
				} else if strings.HasPrefix(rawURL, "https://") || strings.HasPrefix(rawURL, "http://") {
					closeLink()
					result.WriteString(`<a href="` + template.HTMLEscapeString(rawURL) + `" target="_blank" rel="noopener">`)
					linkOpen = true
				}
				// URLs with other schemes (javascript:, data:, etc.) are silently dropped.
			}

		default:
			result.WriteByte(input[i])
			i++
		}
	}

	closeSpan()
	closeLink()
	return result.String()
}
