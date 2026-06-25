package render

import (
	"fmt"
	"math"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gookit/color"
	"github.com/dmitriy/curlstreet/internal/quote"
)

var (
	colorUp     = color.HEX("#5fcf80")
	colorDown   = color.HEX("#e8675f")
	colorDim    = color.HEX("#6f6b64")
	colorFG     = color.HEX("#c2beb6")
	colorHI     = color.HEX("#f1efe9")
	colorBorder = color.HEX("#4c4944")

	badgeGH = color.NewRGBStyle(color.RGB(241, 239, 233), color.RGB(36, 41, 46))  // light on GitHub dark
	badgeLI = color.NewRGBStyle(color.RGB(255, 255, 255), color.RGB(0, 119, 181)) // white on LinkedIn blue
)

// osc8Link wraps text in an OSC 8 hyperlink escape sequence.
// Terminals that support OSC 8 render the text as a clickable link;
// others display the text as-is (the escape sequences are invisible).
func osc8Link(url, text string) string {
	return "\033]8;;" + url + "\033\\" + text + "\033]8;;\033\\"
}

func RenderText(quotes []quote.QuoteResult, events ...quote.EconEvent) (string, error) {
	var market, tickers []quote.QuoteResult
	for _, qr := range quotes {
		if qr.IsMarket {
			market = append(market, qr)
		} else {
			tickers = append(tickers, qr)
		}
	}

	var sb strings.Builder
	sb.WriteString(renderMarketsSection(market, tickers))
	sb.WriteString(renderOnDeckSection(events))

	if len(tickers) > 0 {
		sb.WriteString("\n")
		sb.WriteString(colorBorder.Sprint(strings.Repeat("═", 71)) + "\n")
		sb.WriteString("\n")

		matched := 0
		for _, qr := range tickers {
			if qr.Quote != nil {
				matched++
			}
		}
		sb.WriteString(colorDim.Sprintf("  %d ticker%s matched", matched, pluralS(matched)) + "\n")

		for _, qr := range tickers {
			sb.WriteString("\n")
			sb.WriteString(renderDetailBlock(qr))
		}
	} else {
		sb.WriteString("\n")
		sb.WriteString(
			colorDim.Sprint("  add tickers") + "  " +
				colorBorder.Sprint("→") + "  " +
				colorFG.Sprint("curl curlstreet.sh/") +
				colorHI.Sprint("AAPL,TSLA") + "\n",
		)
	}

	sb.WriteString("\n")
	sb.WriteString(
		"  " + osc8Link("https://github.com/DmitriyRogo/curlstreet.sh", badgeGH.Sprint(" ☆ curlstreet.sh ")) +
			"   " +
			osc8Link("https://www.linkedin.com/in/dmitriy-rogozhnikov", badgeLI.Sprint(" in dmitriy-rogozhnikov ")) + "\n",
	)

	return sb.String(), nil
}

func renderMarketsSection(market, tickers []quote.QuoteResult) string {
	var sb strings.Builder

	status, now := currentMarketStatus(append(market, tickers...))
	rightSide := marketStatusRight(status, now)
	headerText := "MARKETS"
	pad := 71 - len(headerText) - visibleLen(rightSide)
	if pad < 1 {
		pad = 1
	}
	sb.WriteString(colorHI.Sprint(headerText) + strings.Repeat(" ", pad) + rightSide + "\n")
	sb.WriteString("\n")

	if len(market) == 0 {
		sb.WriteString(colorDim.Sprint("  (market data unavailable)") + "\n")
		return sb.String()
	}

	sep := colorBorder.Sprint(" │ ")
	sb.WriteString(
		"  " +
			colorDim.Sprintf("%-6s", "SYM") + sep +
			colorDim.Sprintf("%-20s", "NAME") + sep +
			colorDim.Sprintf("%10s", "LAST") + sep +
			colorDim.Sprintf("%6s", "CHG") + sep +
			colorDim.Sprint("52-WK RANGE") + "\n",
	)
	sb.WriteString("  " + colorBorder.Sprint(strings.Repeat("─", 69)) + "\n")

	for _, qr := range market {
		if qr.Err != nil {
			sb.WriteString(colorDim.Sprintf("  %-6s  %-20s  N/A\n", qr.Err.Symbol, ""))
			continue
		}
		sb.WriteString(renderMarketRow(qr.Quote))
	}

	return sb.String()
}

func renderMarketRow(q *quote.Quote) string {
	var indicator string
	var rawChg string
	switch {
	case q.Change > 0:
		indicator = colorUp.Sprint("▲ ")
		rawChg = fmt.Sprintf("+%.2f%%", q.ChangePercent)
	case q.Change < 0:
		indicator = colorDown.Sprint("▼ ")
		rawChg = fmt.Sprintf("%.2f%%", q.ChangePercent)
	default:
		indicator = "  "
		rawChg = fmt.Sprintf("%.2f%%", q.ChangePercent)
	}

	paddedChg := fmt.Sprintf("%6s", rawChg)
	var chgStr string
	switch {
	case q.Change > 0:
		chgStr = colorUp.Sprint(paddedChg)
	case q.Change < 0:
		chgStr = colorDown.Sprint(paddedChg)
	default:
		chgStr = colorDim.Sprint(paddedChg)
	}

	sym := colorHI.Sprintf("%-6s", q.Symbol)

	name := q.Name
	if len([]rune(name)) > 20 {
		r := []rune(name)
		name = string(r[:17]) + "..."
	}
	nameStr := colorDim.Sprintf("%-20s", name)

	priceStr := fmt.Sprintf("%10s", formatWithCommas(q.Price))
	sep := colorBorder.Sprint(" │ ")

	var rangeStr string
	if q.High52W != nil && q.Low52W != nil {
		lo, hi := *q.Low52W, *q.High52W
		loStr := colorFG.Sprint(" " + shortNum(lo) + " ")
		hiStr := " " + shortNum(hi)
		bar := renderBar(q.Price, lo, hi, 7, q.Change)
		rangeStr = sep + loStr + colorBorder.Sprint("▏") + bar + colorBorder.Sprint("▕") + hiStr
	}

	return indicator + sym + sep + nameStr + sep + priceStr + sep + chgStr + rangeStr + "\n"
}

func renderDetailBlock(qr quote.QuoteResult) string {
	if qr.Err != nil {
		return colorDim.Sprintf("  %s: %s\n", qr.Err.Symbol, qr.Err.Message)
	}
	q := qr.Quote

	var sb strings.Builder

	sb.WriteString(colorHI.Sprint(q.Symbol) + colorDim.Sprint("  ·  ") + colorFG.Sprint(q.Name) + "\n")
	sb.WriteString(colorBorder.Sprint(strings.Repeat("─", 52)) + "\n")

	sb.WriteString(colorDim.Sprint("Last price  ") + formatPriceValue(q) + " " + colorDim.Sprint(q.Currency) + "\n")
	sb.WriteString(colorDim.Sprint("Change      ") + formatChangeDetail(q.Change, q.ChangePercent) + "\n")

	rangeLabel := "52-wk range"
	if q.AssetType == quote.AssetTypeCrypto {
		rangeLabel = "24h range  "
	}
	if q.High52W != nil && q.Low52W != nil {
		lo, hi := *q.Low52W, *q.High52W
		loStr := formatFloat(lo, q)
		hiStr := formatFloat(hi, q)
		bar := renderBar(q.Price, lo, hi, 9, q.Change)
		sb.WriteString(colorDim.Sprint(rangeLabel+" ") + loStr + " " + bar + " " + hiStr + "\n")
	}

	if q.MarketCap != nil {
		sb.WriteString(colorDim.Sprint("Mkt cap     ") + colorFG.Sprint("$"+largeNum(*q.MarketCap)) + "\n")
	}

	secType := securityTypeLabel(q.SecurityType)
	if q.AssetType == quote.AssetTypeCrypto {
		secType = "Cryptocurrency"
	}
	sb.WriteString(colorDim.Sprint("Security    ") + colorFG.Sprint(secType) + "\n")

	exchangeDisplay := q.Exchange
	if q.ExchangeCode != "" {
		exchangeDisplay = q.Exchange + colorDim.Sprint(" · ") + q.ExchangeCode
	}
	if exchangeDisplay != "" {
		sb.WriteString(colorDim.Sprint("Exchange    ") + colorFG.Sprint(exchangeDisplay) + "\n")
	}

	sb.WriteString(colorDim.Sprint("Currency    ") + colorFG.Sprint(q.Currency) + "\n")

	if q.Sector != "" {
		sb.WriteString(colorDim.Sprint("Sector      ") + colorFG.Sprint(q.Sector) + "\n")
	}
	if q.Industry != "" {
		sb.WriteString(colorDim.Sprint("Industry    ") + colorFG.Sprint(q.Industry) + "\n")
	}

	if !q.UpdatedAt.IsZero() {
		loc, _ := time.LoadLocation("America/New_York")
		if loc == nil {
			loc = time.UTC
		}
		sb.WriteString(colorDim.Sprint("Updated     ") + colorDim.Sprint(q.UpdatedAt.In(loc).Format("15:04 MST")) + "\n")
	}

	return sb.String()
}

func renderOnDeckSection(events []quote.EconEvent) string {
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(colorBorder.Sprint(strings.Repeat("═", 53)) + "\n")
	sb.WriteString("\n")

	headerText := "ON DECK"
	rightText := "economic calendar"
	pad := 53 - len(headerText) - len(rightText)
	if pad < 1 {
		pad = 1
	}
	sb.WriteString(colorHI.Sprint(headerText) + strings.Repeat(" ", pad) + colorDim.Sprint(rightText) + "\n")
	sb.WriteString("\n")

	sep := colorBorder.Sprint(" │ ")
	sb.WriteString(
		"    " +
			colorDim.Sprintf("%-16s", "EVENT") +
			sep +
			colorDim.Sprintf("%-19s", "WHEN") +
			sep +
			colorDim.Sprint("IMPACT") + "\n",
	)
	sb.WriteString("  " + colorBorder.Sprint(strings.Repeat("─", 57)) + "\n")

	if len(events) > 0 {
		for _, ev := range events {
			name := []rune(ev.Name)
			nameStr := string(name)
			if len(name) > 16 {
				nameStr = string(name[:15]) + "…"
			}
			impactColor := colorDim
			switch ev.Impact {
			case "high":
				impactColor = colorDown
			case "medium":
				impactColor = colorFG
			}
			sb.WriteString(
				"  " + impactColor.Sprint("▸ ") +
					colorHI.Sprintf("%-16s", nameStr) +
					sep +
					fmt.Sprintf("%-19s", ev.When) +
					sep +
					impactColor.Sprint(ev.Impact) + "\n",
			)
		}
	} else {
		for _, ev := range staticEconEvents() {
			sb.WriteString(
				"  " + colorBorder.Sprint("▸ ") +
					colorHI.Sprintf("%-16s", ev.name) +
					sep +
					fmt.Sprintf("%-19s", ev.when) +
					sep +
					colorDim.Sprint(ev.desc) + "\n",
			)
		}
	}

	return sb.String()
}

// staticEconEvents returns illustrative economic calendar placeholders used
// when the live Finnhub calendar fetch is unavailable.
func staticEconEvents() []struct{ name, when, desc string } {
	return []struct{ name, when, desc string }{
		{"CPI", "Thu 08:30", "inflation print"},
		{"Retail Sales", "Fri 08:30", "consumer demand"},
		{"Jobless Claims", "Thu 08:30", "weekly claims"},
		{"FOMC Minutes", "Wed 14:00", "policy outlook"},
	}
}

// currentMarketStatus returns the US equity market status. It prefers the
// status from a fetched quote (which reflects holidays and early closes) and
// falls back to clock-based computation only when no quote data is available.
func currentMarketStatus(tickers []quote.QuoteResult) (string, time.Time) {
	loc, _ := time.LoadLocation("America/New_York")
	if loc == nil {
		loc = time.UTC
	}
	now := time.Now().In(loc)

	for _, qr := range tickers {
		if qr.Quote != nil && qr.Quote.MarketStatus != nil {
			return *qr.Quote.MarketStatus, now
		}
	}

	// Fallback: derive from clock when no live data is present.
	wd := now.Weekday()
	if wd == time.Saturday || wd == time.Sunday {
		return quote.MarketStatusClosed, now
	}
	total := now.Hour()*60 + now.Minute()
	const openMin = 9*60 + 30
	const closeMin = 16 * 60
	const preMin = 4 * 60
	const ahEnd = 20 * 60
	switch {
	case total >= openMin && total < closeMin:
		return quote.MarketStatusOpen, now
	case total >= preMin && total < openMin:
		return quote.MarketStatusPreMarket, now
	case total >= closeMin && total < ahEnd:
		return quote.MarketStatusAfterHours, now
	default:
		return quote.MarketStatusClosed, now
	}
}

func marketStatusRight(status string, now time.Time) string {
	timeStr := now.Format("15:04 MST")
	switch status {
	case quote.MarketStatusOpen:
		close_ := time.Date(now.Year(), now.Month(), now.Day(), 16, 0, 0, 0, now.Location())
		rem := close_.Sub(now)
		h := int(rem.Hours())
		m := int(rem.Minutes()) % 60
		return colorUp.Sprint("● OPEN") +
			colorDim.Sprintf(" · %s · closes %dh %dm", timeStr, h, m)
	case quote.MarketStatusPreMarket:
		open_ := time.Date(now.Year(), now.Month(), now.Day(), 9, 30, 0, 0, now.Location())
		rem := open_.Sub(now)
		h := int(rem.Hours())
		m := int(rem.Minutes()) % 60
		return colorFG.Sprint("● PRE-MARKET") +
			colorDim.Sprintf(" · %s · opens %dh %dm", timeStr, h, m)
	case quote.MarketStatusAfterHours:
		return colorFG.Sprint("● AFTER HOURS") + colorDim.Sprint(" · "+timeStr)
	default:
		return colorDown.Sprint("● CLOSED") + colorDim.Sprint(" · "+timeStr)
	}
}

func renderBar(price, lo, hi float64, width int, change float64) string {
	var pos int
	if hi > lo {
		ratio := (price - lo) / (hi - lo)
		if ratio < 0 {
			ratio = 0
		}
		if ratio > 1 {
			ratio = 1
		}
		pos = int(math.Round(ratio * float64(width-1)))
	}

	var dot string
	switch {
	case change > 0:
		dot = colorUp.Sprint("●")
	case change < 0:
		dot = colorDown.Sprint("●")
	default:
		dot = colorFG.Sprint("●")
	}

	left := colorBorder.Sprint(strings.Repeat("─", pos))
	right := colorBorder.Sprint(strings.Repeat("─", width-1-pos))
	return left + dot + right
}

func formatPriceValue(q *quote.Quote) string {
	if q.AssetType == quote.AssetTypeCrypto && q.Price < 0.01 {
		return colorHI.Sprintf("%.8f", q.Price)
	}
	return colorHI.Sprintf("%.2f", q.Price)
}

func formatPriceCompact(q *quote.Quote) string {
	if q.AssetType == quote.AssetTypeCrypto && q.Price < 0.01 {
		return fmt.Sprintf("%.8f", q.Price)
	}
	return fmt.Sprintf("%.2f", q.Price)
}

func formatFloat(v float64, q *quote.Quote) string {
	if q.AssetType == quote.AssetTypeCrypto && v < 0.01 {
		return fmt.Sprintf("%.8f", v)
	}
	return fmt.Sprintf("%.2f", v)
}

func formatChangeDetail(change, pct float64) string {
	var arrow, s string
	if change > 0 {
		arrow = colorUp.Sprint("▲ ")
		s = colorUp.Sprintf("+%.2f  (+%.2f%%)", change, pct)
	} else if change < 0 {
		arrow = colorDown.Sprint("▼ ")
		s = colorDown.Sprintf("%.2f  (%.2f%%)", change, pct)
	} else {
		arrow = "  "
		s = colorDim.Sprintf("%.2f  (%.2f%%)", change, pct)
	}
	return arrow + s
}

func securityTypeLabel(t string) string {
	switch t {
	case "EQS", "EQ":
		return "Common Stock"
	case "ETF":
		return "ETF"
	case "ADR":
		return "ADR"
	case "REIT":
		return "REIT"
	case "WARRANT":
		return "Warrant"
	default:
		if t == "" {
			return "Common Stock"
		}
		return t
	}
}

func formatWithCommas(price float64) string {
	s := fmt.Sprintf("%.2f", price)
	dot := strings.Index(s, ".")
	intPart := s
	decPart := ""
	if dot >= 0 {
		intPart = s[:dot]
		decPart = s[dot:]
	}
	var out strings.Builder
	for i, c := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			out.WriteRune(',')
		}
		out.WriteRune(c)
	}
	return out.String() + decPart
}

func shortNum(v float64) string {
	switch {
	case v >= 1_000_000:
		return fmt.Sprintf("%.0fM", v/1_000_000)
	case v >= 10_000:
		return fmt.Sprintf("%.0fk", v/1_000)
	case v >= 1_000:
		return fmt.Sprintf("%.1fk", v/1_000)
	default:
		return fmt.Sprintf("%.0f", v)
	}
}

// largeNum formats large integers with T/B/M/K suffixes for compact display.
func largeNum(v int64) string {
	switch {
	case v >= 1_000_000_000_000:
		return fmt.Sprintf("%.2fT", float64(v)/1_000_000_000_000)
	case v >= 1_000_000_000:
		return fmt.Sprintf("%.2fB", float64(v)/1_000_000_000)
	case v >= 1_000_000:
		return fmt.Sprintf("%.2fM", float64(v)/1_000_000)
	case v >= 1_000:
		return fmt.Sprintf("%.2fK", float64(v)/1_000)
	default:
		return fmt.Sprintf("%d", v)
	}
}

func pluralS(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// visibleLen returns the number of visible runes in a string containing ANSI
// SGR and OSC 8 escape sequences.
func visibleLen(s string) int {
	var plain strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == 0x1b && i+1 < len(s) {
			switch s[i+1] {
			case '[':
				// SGR: skip to 'm'
				i += 2
				for i < len(s) && s[i] != 'm' {
					i++
				}
				if i < len(s) {
					i++
				}
			case ']':
				// OSC: skip to BEL or ST
				i += 2
				for i < len(s) {
					if s[i] == 0x07 {
						i++
						break
					}
					if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '\\' {
						i += 2
						break
					}
					i++
				}
			default:
				plain.WriteByte(s[i])
				i++
			}
		} else {
			plain.WriteByte(s[i])
			i++
		}
	}
	return utf8.RuneCountInString(plain.String())
}

// formatChange and formatVolume kept for test compatibility.
func formatChange(change, pct float64) string {
	s := fmt.Sprintf("%+.2f (%+.2f%%)", change, pct)
	if change > 0 {
		return color.Green.Sprint(s)
	} else if change < 0 {
		return color.Red.Sprint(s)
	}
	return s
}

func formatVolume(v int64) string {
	switch {
	case v >= 1_000_000_000:
		return fmt.Sprintf("%.2fB", float64(v)/1_000_000_000)
	case v >= 1_000_000:
		return fmt.Sprintf("%.2fM", float64(v)/1_000_000)
	case v >= 1_000:
		return fmt.Sprintf("%.2fK", float64(v)/1_000)
	default:
		return fmt.Sprintf("%d", v)
	}
}
