package app_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gregLibert/go-trend-notifier/internal/app"
	"github.com/gregLibert/go-trend-notifier/internal/config"
	"github.com/gregLibert/go-trend-notifier/internal/fetch"
	"github.com/gregLibert/go-trend-notifier/internal/market"
	"github.com/gregLibert/go-trend-notifier/internal/notify"
)

type fixedProvider struct {
	bars []market.DailyBar
}

func (f *fixedProvider) Name() string { return "fixed" }

func (f *fixedProvider) FetchDailyBars(context.Context, string) ([]market.DailyBar, error) {
	return f.bars, nil
}

type failingProvider struct{}

func (f *failingProvider) Name() string { return "failing" }

func (f *failingProvider) FetchDailyBars(context.Context, string) ([]market.DailyBar, error) {
	return nil, fmt.Errorf("api timeout")
}

func makeBars(count int, close, low float64) []market.DailyBar {
	start := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
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

func newTestTelegram(t *testing.T) (*notify.TelegramClient, *atomic.Int32) {
	t.Helper()
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	telegram, err := notify.NewTelegramClient("tok", notify.WithTelegramBaseURL(srv.URL), notify.WithTelegramHTTPClient(srv.Client()))
	if err != nil {
		t.Fatalf("telegram: %v", err)
	}
	return telegram, &calls
}

func TestRunner_Run_SendsWhenAlert(t *testing.T) {
	t.Parallel()

	telegram, calls := newTestTelegram(t)
	bars := makeBars(220, 50, 50)

	cfg := config.Config{
		TelegramToken:   "tok",
		TelegramChatIDs: []string{"1"},
		Assets:          []config.Asset{{Symbol: "TEST", Provider: "fixed"}},
	}

	runner, err := app.NewRunner(cfg, map[string]fetch.Provider{"fixed": &fixedProvider{bars: bars}}, telegram, nil)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	if err := runner.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("telegram calls = %d, want 1", calls.Load())
	}
}

func TestRunner_Run_NoAlertSkipsTelegram(t *testing.T) {
	t.Parallel()

	telegram, calls := newTestTelegram(t)
	bars := makeBars(220, 200, 150)

	cfg := config.Config{
		TelegramChatIDs: []string{"1"},
		Assets:          []config.Asset{{Symbol: "TEST", Provider: "fixed"}},
	}

	runner, err := app.NewRunner(cfg, map[string]fetch.Provider{"fixed": &fixedProvider{bars: bars}}, telegram, nil)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	if err := runner.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if calls.Load() != 0 {
		t.Fatalf("telegram calls = %d, want 0", calls.Load())
	}
}

func TestRunner_Run_ConcurrentAssetsNoRace(t *testing.T) {
	t.Parallel()

	telegram, calls := newTestTelegram(t)
	bars := makeBars(220, 50, 50)

	assets := make([]config.Asset, 8)
	for i := range assets {
		assets[i] = config.Asset{Symbol: fmt.Sprintf("SYM%d", i), Provider: "fixed"}
	}

	cfg := config.Config{
		TelegramChatIDs: []string{"1"},
		Assets:          assets,
	}

	runner, err := app.NewRunner(cfg, map[string]fetch.Provider{"fixed": &fixedProvider{bars: bars}}, telegram, nil)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	if err := runner.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("telegram calls = %d, want 1", calls.Load())
	}
}

func TestRunner_Run_PartialFailureStillSendsAlerts(t *testing.T) {
	t.Parallel()

	telegram, calls := newTestTelegram(t)
	bars := makeBars(220, 50, 50)

	cfg := config.Config{
		TelegramChatIDs: []string{"1"},
		Assets: []config.Asset{
			{Symbol: "OK", Provider: "fixed"},
			{Symbol: "BAD", Provider: "failing"},
		},
	}

	providers := map[string]fetch.Provider{
		"fixed":   &fixedProvider{bars: bars},
		"failing": &failingProvider{},
	}

	runner, err := app.NewRunner(cfg, providers, telegram, nil)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	err = runner.Run(context.Background())
	if err == nil {
		t.Fatal("expected aggregate fetch error")
	}
	if calls.Load() != 1 {
		t.Fatalf("telegram calls = %d, want 1", calls.Load())
	}
}

func TestRunner_Run_AllFetchesFail(t *testing.T) {
	t.Parallel()

	telegram, calls := newTestTelegram(t)

	cfg := config.Config{
		TelegramChatIDs: []string{"1"},
		Assets: []config.Asset{
			{Symbol: "BAD1", Provider: "failing"},
			{Symbol: "BAD2", Provider: "failing"},
		},
	}

	runner, err := app.NewRunner(cfg, map[string]fetch.Provider{"failing": &failingProvider{}}, telegram, nil)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	err = runner.Run(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if calls.Load() != 0 {
		t.Fatalf("telegram calls = %d, want 0", calls.Load())
	}
}
