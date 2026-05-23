package market

import (
	"fmt"
	"html"
	"strings"
	"time"
)

// FormatAlertMessage builds a consolidated Telegram HTML message.
func FormatAlertMessage(alerts []AssetAlert, generatedAt time.Time) string {
	var b strings.Builder
	b.WriteString("<b>Market trend alerts</b>\n")
	fmt.Fprintf(&b, "<i>Generated: %s UTC</i>\n\n", html.EscapeString(generatedAt.UTC().Format(time.RFC3339)))

	for _, a := range alerts {
		writeAssetAlert(&b, a)
	}
	return strings.TrimSpace(b.String())
}

func writeAssetAlert(b *strings.Builder, a AssetAlert) {
	s := a.Snapshot
	fmt.Fprintf(b, "<b>%s</b> (%s)\n", html.EscapeString(s.Symbol), html.EscapeString(s.Source))
	fmt.Fprintf(b, "Last bar: %s\n", html.EscapeString(s.LastBarDate.Format("2006-01-02")))
	fmt.Fprintf(b, "Close: <code>%.4f</code>\n", s.Current)
	fmt.Fprintf(b, "MA50: <code>%.4f</code> | MA200: <code>%.4f</code>\n", s.MA50, s.MA200)
	fmt.Fprintf(b, "50d low: <code>%.4f</code> | 200d low: <code>%.4f</code>\n", s.Low50, s.Low200)
	b.WriteString("Triggered:\n")
	for _, c := range a.Conditions {
		fmt.Fprintf(b, "  - %s\n", html.EscapeString(ConditionLabel(c)))
	}
	b.WriteString("\n")
}
