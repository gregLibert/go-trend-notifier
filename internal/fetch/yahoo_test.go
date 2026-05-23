package fetch_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gregLibert/go-trend-notifier/internal/fetch"
)

const sampleYahooChart = `{
  "chart": {
    "result": [{
      "timestamp": [1704067200, 1704153600],
      "indicators": {
        "quote": [{
          "close": [100.0, 101.5],
          "low": [95.0, 96.0]
        }]
      }
    }]
  }
}`

func TestYahooClient_FetchDailyBars(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v8/finance/chart/LQQ.PA" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(sampleYahooChart))
	}))
	defer srv.Close()

	client := fetch.NewYahooClient(
		fetch.WithYahooBaseURL(srv.URL),
		fetch.WithYahooHTTPClient(srv.Client()),
	)

	bars, err := client.FetchDailyBars(context.Background(), "LQQ.PA")
	if err != nil {
		t.Fatalf("FetchDailyBars: %v", err)
	}
	if len(bars) != 2 {
		t.Fatalf("got %d bars, want 2", len(bars))
	}
	if bars[1].Close != 101.5 {
		t.Fatalf("second close = %v, want 101.5", bars[1].Close)
	}
}

func TestYahooClient_APIError(t *testing.T) {
	t.Parallel()

	body := `{"chart":{"result":[],"error":{"description":"Invalid symbol"}}}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	client := fetch.NewYahooClient(
		fetch.WithYahooBaseURL(srv.URL),
		fetch.WithYahooHTTPClient(srv.Client()),
	)

	_, err := client.FetchDailyBars(context.Background(), "BAD")
	if err == nil {
		t.Fatal("expected error")
	}
}
