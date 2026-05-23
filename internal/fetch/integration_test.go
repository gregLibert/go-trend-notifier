//go:build integration

package fetch

import (
	"context"
	"testing"
	"time"

	"github.com/gregLibert/go-trend-notifier/internal/market"
)

const (
	integrationTimeout = 2 * time.Minute
	maxLastBarAge      = 7 * 24 * time.Hour
)

type liveFetcher interface {
	FetchDailyBars(ctx context.Context, symbol string) ([]market.DailyBar, error)
}

func TestLiveFetchDailyBars(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout)
	defer cancel()

	tests := []struct {
		name      string
		symbol    string
		newClient func() liveFetcher
	}{
		{
			name:   "binance",
			symbol: "BTCUSDT",
			newClient: func() liveFetcher {
				return NewBinanceClient()
			},
		},
		{
			name:   "yahoo",
			symbol: "LQQ.PA",
			newClient: func() liveFetcher {
				return NewYahooClient()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := tt.newClient()
			bars, err := client.FetchDailyBars(ctx, tt.symbol)
			if err != nil {
				t.Fatalf("FetchDailyBars(%q): %v", tt.symbol, err)
			}
			assertLiveBars(t, tt.name, tt.symbol, bars)
		})
	}
}

func assertLiveBars(t *testing.T, provider, symbol string, bars []market.DailyBar) {
	t.Helper()

	if len(bars) == 0 {
		t.Fatalf("%s %s: expected non-empty bars", provider, symbol)
	}

	last := bars[len(bars)-1]
	if last.Close <= 0 {
		t.Fatalf("%s %s: last close must be positive, got %v", provider, symbol, last.Close)
	}
	if last.Low <= 0 {
		t.Fatalf("%s %s: last low must be positive, got %v", provider, symbol, last.Low)
	}

	age := time.Now().UTC().Sub(last.Date.UTC())
	if age > maxLastBarAge {
		t.Fatalf(
			"%s %s: last bar date %s is stale (age %s, max %s)",
			provider,
			symbol,
			last.Date.UTC().Format(time.RFC3339),
			age.Truncate(time.Second),
			maxLastBarAge,
		)
	}

	t.Logf(
		"%s %s: %d bars, last=%s close=%.4f",
		provider,
		symbol,
		len(bars),
		last.Date.UTC().Format("2006-01-02"),
		last.Close,
	)
}
