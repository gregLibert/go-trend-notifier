package market_test

import (
	"testing"
	"time"

	"github.com/gregLibert/go-trend-notifier/internal/market"
)

func TestComputeSnapshot(t *testing.T) {
	t.Parallel()

	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	flat := makeDailyBars(base, 220, 100, 95)
	declining := makeDecliningBars(base, 220)

	tests := []struct {
		name    string
		bars    []market.DailyBar
		wantErr bool
	}{
		{name: "insufficient bars", bars: flat[:10], wantErr: true},
		{name: "flat series", bars: flat, wantErr: false},
		{name: "declining series computes", bars: declining, wantErr: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := market.ComputeSnapshot("TEST", "unit", tt.bars)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.name == "flat series" {
				if got.MA50 != 100 {
					t.Fatalf("MA50 = %v, want 100", got.MA50)
				}
				if got.Low50 != 95 {
					t.Fatalf("Low50 = %v, want 95", got.Low50)
				}
			}
			if got.Current != tt.bars[len(tt.bars)-1].Close {
				t.Fatalf("Current = %v, want %v", got.Current, tt.bars[len(tt.bars)-1].Close)
			}
		})
	}
}

func makeDailyBars(start time.Time, count int, close, low float64) []market.DailyBar {
	bars := make([]market.DailyBar, count)
	for i := 0; i < count; i++ {
		bars[i] = market.DailyBar{
			Date:  start.AddDate(0, 0, i),
			Close: close,
			Low:   low,
		}
	}
	return bars
}

func makeDecliningBars(start time.Time, count int) []market.DailyBar {
	bars := make([]market.DailyBar, count)
	for i := 0; i < count; i++ {
		price := float64(300 - i)
		bars[i] = market.DailyBar{
			Date:  start.AddDate(0, 0, i),
			Close: price,
			Low:   price - 5,
		}
	}
	return bars
}
