package market

import "fmt"

// Condition describes a single breached threshold.
type Condition string

const (
	ConditionBelowMA50       Condition = "price_below_ma50"
	ConditionBelowMA200      Condition = "price_below_ma200"
	ConditionAtOrBelowLow50  Condition = "price_at_or_below_50d_low"
	ConditionAtOrBelowLow200 Condition = "price_at_or_below_200d_low"
)

// AssetAlert groups triggered conditions for one asset.
type AssetAlert struct {
	Snapshot   Snapshot
	Conditions []Condition
}

// EvaluateConditions returns all alert conditions met by the latest close.
func EvaluateConditions(s Snapshot) []Condition {
	var out []Condition
	if s.Current < s.MA50 {
		out = append(out, ConditionBelowMA50)
	}
	if s.Current < s.MA200 {
		out = append(out, ConditionBelowMA200)
	}
	if s.Current <= s.Low50 {
		out = append(out, ConditionAtOrBelowLow50)
	}
	if s.Current <= s.Low200 {
		out = append(out, ConditionAtOrBelowLow200)
	}
	return out
}

// ConditionLabel returns a human-readable label for Markdown output.
func ConditionLabel(c Condition) string {
	switch c {
	case ConditionBelowMA50:
		return "Current price is below the 50-day moving average"
	case ConditionBelowMA200:
		return "Current price is below the 200-day moving average"
	case ConditionAtOrBelowLow50:
		return "Current price is at or below the 50-day low"
	case ConditionAtOrBelowLow200:
		return "Current price is at or below the 200-day low"
	default:
		return fmt.Sprintf("unknown condition: %s", c)
	}
}
