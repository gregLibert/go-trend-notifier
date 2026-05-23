package market

import "time"

// DailyBar is one trading day's OHLCV snapshot.
type DailyBar struct {
	Date  time.Time
	Close float64
	Low   float64
}

// Snapshot holds the latest price and computed indicators for alerting.
type Snapshot struct {
	Symbol      string
	Source      string
	Current     float64
	MA50        float64
	MA200       float64
	Low50       float64
	Low200      float64
	LastBarDate time.Time
}
