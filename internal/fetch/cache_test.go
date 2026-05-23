package fetch_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gregLibert/go-trend-notifier/internal/fetch"
	"github.com/gregLibert/go-trend-notifier/internal/market"
)

type stubProvider struct {
	name  string
	calls int
	bars  []market.DailyBar
	err   error
}

func (s *stubProvider) Name() string { return s.name }

func (s *stubProvider) FetchDailyBars(context.Context, string) ([]market.DailyBar, error) {
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	return s.bars, nil
}

func TestCachedProvider_UsesCacheOnSecondCall(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	inner := &stubProvider{
		name: "stub",
		bars: []market.DailyBar{{Date: time.Now(), Close: 1, Low: 1}},
	}

	cached, err := fetch.NewCachedProvider(inner, dir, time.Hour, nil)
	if err != nil {
		t.Fatalf("NewCachedProvider: %v", err)
	}

	ctx := context.Background()
	if _, err := cached.FetchDailyBars(ctx, "SYM"); err != nil {
		t.Fatalf("first fetch: %v", err)
	}
	if _, err := cached.FetchDailyBars(ctx, "SYM"); err != nil {
		t.Fatalf("second fetch: %v", err)
	}
	if inner.calls != 1 {
		t.Fatalf("inner calls = %d, want 1", inner.calls)
	}
}

func TestCachedProvider_RefetchAfterTTL(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	inner := &stubProvider{
		name: "stub",
		bars: []market.DailyBar{{Date: time.Now(), Close: 1, Low: 1}},
	}

	cached, err := fetch.NewCachedProvider(inner, dir, time.Millisecond, nil)
	if err != nil {
		t.Fatalf("NewCachedProvider: %v", err)
	}

	ctx := context.Background()
	if _, err := cached.FetchDailyBars(ctx, "SYM"); err != nil {
		t.Fatalf("first fetch: %v", err)
	}
	time.Sleep(5 * time.Millisecond)
	if _, err := cached.FetchDailyBars(ctx, "SYM"); err != nil {
		t.Fatalf("second fetch: %v", err)
	}
	if inner.calls != 2 {
		t.Fatalf("inner calls = %d, want 2", inner.calls)
	}
}

func TestCachedProvider_PropagatesError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	inner := &stubProvider{name: "stub", err: errors.New("boom")}

	cached, err := fetch.NewCachedProvider(inner, dir, time.Hour, nil)
	if err != nil {
		t.Fatalf("NewCachedProvider: %v", err)
	}

	_, err = cached.FetchDailyBars(context.Background(), "SYM")
	if err == nil {
		t.Fatal("expected error")
	}
}
