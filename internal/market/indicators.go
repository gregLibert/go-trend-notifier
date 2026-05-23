package market

import (
	"fmt"
	"math"
)

const (
	windowMA50  = 50
	windowMA200 = 200
)

// ComputeSnapshot derives indicators from daily bars. The last bar is treated as the current session.
func ComputeSnapshot(symbol, source string, bars []DailyBar) (Snapshot, error) {
	if len(bars) == 0 {
		return Snapshot{}, fmt.Errorf("market: no bars for %s", symbol)
	}
	if len(bars) < windowMA200 {
		return Snapshot{}, fmt.Errorf("market: %s needs at least %d bars, got %d", symbol, windowMA200, len(bars))
	}

	last := bars[len(bars)-1]
	ma50, err := movingAverage(bars, windowMA50)
	if err != nil {
		return Snapshot{}, err
	}
	ma200, err := movingAverage(bars, windowMA200)
	if err != nil {
		return Snapshot{}, err
	}
	low50, err := periodLow(bars, windowMA50)
	if err != nil {
		return Snapshot{}, err
	}
	low200, err := periodLow(bars, windowMA200)
	if err != nil {
		return Snapshot{}, err
	}

	return Snapshot{
		Symbol:      symbol,
		Source:      source,
		Current:     last.Close,
		MA50:        ma50,
		MA200:       ma200,
		Low50:       low50,
		Low200:      low200,
		LastBarDate: last.Date,
	}, nil
}

func movingAverage(bars []DailyBar, window int) (float64, error) {
	if len(bars) < window {
		return 0, fmt.Errorf("market: moving average window %d exceeds %d bars", window, len(bars))
	}
	slice := bars[len(bars)-window:]
	var sum float64
	for _, b := range slice {
		sum += b.Close
	}
	return sum / float64(window), nil
}

func periodLow(bars []DailyBar, window int) (float64, error) {
	if len(bars) < window {
		return 0, fmt.Errorf("market: low window %d exceeds %d bars", window, len(bars))
	}
	slice := bars[len(bars)-window:]
	low := math.MaxFloat64
	for _, b := range slice {
		if b.Low < low {
			low = b.Low
		}
	}
	return low, nil
}
