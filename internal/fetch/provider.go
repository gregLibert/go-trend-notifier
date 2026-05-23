package fetch

import (
	"context"

	"github.com/gregLibert/go-trend-notifier/internal/market"
)

// Provider fetches daily OHLCV history for a symbol.
type Provider interface {
	Name() string
	FetchDailyBars(ctx context.Context, symbol string) ([]market.DailyBar, error)
}
