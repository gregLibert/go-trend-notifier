package market_test

import (
	"strings"
	"testing"
	"time"

	"github.com/gregLibert/go-trend-notifier/internal/market"
)

func TestFormatAlertMessage(t *testing.T) {
	t.Parallel()

	when := time.Date(2025, 5, 23, 12, 0, 0, 0, time.UTC)
	alerts := []market.AssetAlert{{
		Snapshot: market.Snapshot{
			Symbol:      "BTCUSDT",
			Source:      "binance",
			Current:     90,
			MA50:        100,
			MA200:       95,
			Low50:       88,
			Low200:      85,
			LastBarDate: when,
		},
		Conditions: []market.Condition{market.ConditionBelowMA50},
	}}

	msg := market.FormatAlertMessage(alerts, when)
	checks := []string{
		"<b>Market trend alerts</b>",
		"<i>Generated:",
		"<b>BTCUSDT</b>",
		"binance",
		"<code>90.0000</code>",
		"below the 50-day moving average",
	}
	for _, want := range checks {
		if !strings.Contains(msg, want) {
			t.Fatalf("message missing %q:\n%s", want, msg)
		}
	}
	if strings.Contains(msg, "*") || strings.Contains(msg, "`") {
		t.Fatalf("message still contains Markdown markers:\n%s", msg)
	}
}

func TestFormatAlertMessage_EscapesHTMLInSymbol(t *testing.T) {
	t.Parallel()

	when := time.Date(2025, 5, 23, 12, 0, 0, 0, time.UTC)
	alerts := []market.AssetAlert{{
		Snapshot: market.Snapshot{
			Symbol:      "FOO<BAR>&",
			Source:      "yahoo",
			Current:     1,
			MA50:        2,
			MA200:       2,
			Low50:       1,
			Low200:      1,
			LastBarDate: when,
		},
		Conditions: []market.Condition{market.ConditionBelowMA50},
	}}

	msg := market.FormatAlertMessage(alerts, when)
	if strings.Contains(msg, "<BAR>") {
		t.Fatalf("unescaped symbol in message:\n%s", msg)
	}
	if !strings.Contains(msg, "FOO&lt;BAR&gt;&amp;") {
		t.Fatalf("expected escaped symbol in message:\n%s", msg)
	}
}
