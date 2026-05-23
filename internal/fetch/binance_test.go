package fetch_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gregLibert/go-trend-notifier/internal/fetch"
)

func TestBinanceClient_FetchDailyBars(t *testing.T) {
	t.Parallel()

	openTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	rows := [][]any{
		{openTime, "1", "2", "0.5", "1.5", "10"},
		{openTime + 86_400_000, "1.5", "2", "1.0", "2.0", "10"},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/klines" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(rows)
	}))
	defer srv.Close()

	client := fetch.NewBinanceClient(
		fetch.WithBinanceBaseURL(srv.URL),
		fetch.WithBinanceHTTPClient(srv.Client()),
		fetch.WithBinanceClock(func() time.Time {
			return time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC)
		}),
	)

	bars, err := client.FetchDailyBars(context.Background(), "BTCUSDT")
	if err != nil {
		t.Fatalf("FetchDailyBars: %v", err)
	}
	if len(bars) != 2 {
		t.Fatalf("got %d bars, want 2", len(bars))
	}
	if bars[0].Close != 1.5 {
		t.Fatalf("first close = %v, want 1.5", bars[0].Close)
	}
	if bars[0].Low != 0.5 {
		t.Fatalf("first low = %v, want 0.5", bars[0].Low)
	}
}

func TestBinanceClient_HTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "rate limited", http.StatusTeapot)
	}))
	defer srv.Close()

	client := fetch.NewBinanceClient(
		fetch.WithBinanceBaseURL(srv.URL),
		fetch.WithBinanceHTTPClient(srv.Client()),
	)

	_, err := client.FetchDailyBars(context.Background(), "ETHUSDT")
	if err == nil {
		t.Fatal("expected error")
	}
}
