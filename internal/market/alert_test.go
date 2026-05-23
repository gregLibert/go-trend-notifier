package market_test

import (
	"slices"
	"testing"

	"github.com/gregLibert/go-trend-notifier/internal/market"
)

func TestEvaluateConditions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		snap market.Snapshot
		want []market.Condition
	}{
		{
			name: "no alerts",
			snap: market.Snapshot{Current: 110, MA50: 100, MA200: 90, Low50: 80, Low200: 70},
			want: nil,
		},
		{
			name: "below ma50 only",
			snap: market.Snapshot{Current: 95, MA50: 100, MA200: 90, Low50: 80, Low200: 70},
			want: []market.Condition{market.ConditionBelowMA50},
		},
		{
			name: "below ma200",
			snap: market.Snapshot{Current: 85, MA50: 100, MA200: 90, Low50: 80, Low200: 70},
			want: []market.Condition{market.ConditionBelowMA50, market.ConditionBelowMA200},
		},
		{
			name: "at 50d low",
			snap: market.Snapshot{Current: 80, MA50: 100, MA200: 90, Low50: 80, Low200: 70},
			want: []market.Condition{
				market.ConditionBelowMA50,
				market.ConditionBelowMA200,
				market.ConditionAtOrBelowLow50,
			},
		},
		{
			name: "all conditions",
			snap: market.Snapshot{Current: 70, MA50: 100, MA200: 90, Low50: 80, Low200: 70},
			want: []market.Condition{
				market.ConditionBelowMA50,
				market.ConditionBelowMA200,
				market.ConditionAtOrBelowLow50,
				market.ConditionAtOrBelowLow200,
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := market.EvaluateConditions(tt.snap)
			if !slices.Equal(got, tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}
