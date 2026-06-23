package render

import (
	"fmt"
	"strings"
	"time"

	"github.com/gookit/color"
	"github.com/dmitriy/curlstreet/internal/quote"
)

func RenderText(quotes []quote.QuoteResult) (string, error) {
	var sb strings.Builder
	divider := strings.Repeat("─", 78)

	for i, qr := range quotes {
		if i > 0 {
			sb.WriteString(divider + "\n")
		}
		if qr.Err != nil {
			sb.WriteString(fmt.Sprintf("Error [%d]: %s\n", qr.Err.Code, qr.Err.Message))
			continue
		}
		sb.WriteString(renderQuoteText(qr.Quote))
	}
	return sb.String(), nil
}

func renderQuoteText(q *quote.Quote) string {
	var sb strings.Builder

	// Header: Symbol + Name
	header := fmt.Sprintf("%-10s  %s", q.Symbol, q.Name)
	if len(header) > 80 {
		header = header[:80]
	}
	sb.WriteString(header + "\n")

	// Price line with color
	changeStr := formatChange(q.Change, q.ChangePercent)
	priceLine := fmt.Sprintf("%-20s  %s", formatPrice(q), changeStr)
	if len(priceLine) > 80 {
		priceLine = priceLine[:80]
	}
	sb.WriteString(priceLine + "\n")

	// High / Low
	highLabel := "52W High"
	lowLabel := "52W Low"
	if q.AssetType == quote.AssetTypeCrypto {
		highLabel = "24h High"
		lowLabel = "24h Low"
	}
	if q.High52W != nil && q.Low52W != nil {
		hl := fmt.Sprintf("%s: %-12s  %s: %s", highLabel, formatFloat(*q.High52W, q), lowLabel, formatFloat(*q.Low52W, q))
		if len(hl) > 80 {
			hl = hl[:80]
		}
		sb.WriteString(hl + "\n")
	}

	// Volume
	if q.Volume != nil {
		volLine := fmt.Sprintf("Volume: %-15s", formatVolume(*q.Volume))
		if q.AvgVolume != nil {
			volLine += fmt.Sprintf("  Avg Volume: %s", formatVolume(*q.AvgVolume))
		}
		if len(volLine) > 80 {
			volLine = volLine[:80]
		}
		sb.WriteString(volLine + "\n")
	}

	// Market status (stocks only)
	if q.AssetType == quote.AssetTypeStock && q.MarketStatus != nil {
		var statusLabel string
		switch *q.MarketStatus {
		case quote.MarketStatusOpen:
			statusLabel = "● LIVE"
		default:
			statusLabel = "● LAST CLOSE"
		}
		sb.WriteString(statusLabel + "\n")
	}

	// Updated at
	updLine := fmt.Sprintf("Updated: %s", q.UpdatedAt.Format(time.RFC3339))
	if len(updLine) > 80 {
		updLine = updLine[:80]
	}
	sb.WriteString(updLine + "\n")

	return sb.String()
}

func formatPrice(q *quote.Quote) string {
	price := q.Price
	if q.AssetType == quote.AssetTypeCrypto && price < 0.01 {
		return fmt.Sprintf("%s %.8f", q.Currency, price)
	}
	return fmt.Sprintf("%s %.2f", q.Currency, price)
}

func formatFloat(v float64, q *quote.Quote) string {
	if q.AssetType == quote.AssetTypeCrypto && v < 0.01 {
		return fmt.Sprintf("%.8f", v)
	}
	return fmt.Sprintf("%.2f", v)
}

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
