package fetch

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gregLibert/go-trend-notifier/internal/market"
)

const (
	binanceKlinesPath   = "/api/v3/klines"
	binanceMaxKlines    = 1000
	binanceInterval1Day = "1d"
	historyYears        = 2
)

// BinanceClient fetches spot daily klines from Binance.
type BinanceClient struct {
	baseURL    string
	httpClient *http.Client
	logger     *slog.Logger
	now        func() time.Time
}

// BinanceOption configures BinanceClient.
type BinanceOption func(*BinanceClient)

// WithBinanceBaseURL overrides the API base URL (useful in tests).
func WithBinanceBaseURL(base string) BinanceOption {
	return func(c *BinanceClient) {
		c.baseURL = base
	}
}

// WithBinanceHTTPClient sets a custom HTTP client.
func WithBinanceHTTPClient(client *http.Client) BinanceOption {
	return func(c *BinanceClient) {
		c.httpClient = client
	}
}

// WithBinanceLogger sets the logger.
func WithBinanceLogger(logger *slog.Logger) BinanceOption {
	return func(c *BinanceClient) {
		c.logger = logger
	}
}

// WithBinanceClock overrides time.Now for tests.
func WithBinanceClock(now func() time.Time) BinanceOption {
	return func(c *BinanceClient) {
		c.now = now
	}
}

// NewBinanceClient creates a Binance spot klines client.
func NewBinanceClient(opts ...BinanceOption) *BinanceClient {
	c := &BinanceClient{
		baseURL:    "https://api.binance.com",
		httpClient: &http.Client{Timeout: 30 * time.Second},
		logger:     slog.Default(),
		now:        time.Now,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *BinanceClient) Name() string {
	return "binance"
}

// FetchDailyBars downloads roughly two years of daily candles.
func (c *BinanceClient) FetchDailyBars(ctx context.Context, symbol string) ([]market.DailyBar, error) {
	start := c.now().UTC().AddDate(-historyYears, 0, 0)
	end := c.now().UTC()

	var all []market.DailyBar
	cursor := start
	for cursor.Before(end) {
		chunk, err := c.fetchKlines(ctx, symbol, cursor, end)
		if err != nil {
			return nil, err
		}
		if len(chunk) == 0 {
			break
		}
		all = append(all, chunk...)
		last := chunk[len(chunk)-1].Date
		cursor = last.Add(24 * time.Hour)
		if len(chunk) < binanceMaxKlines {
			break
		}
	}
	return dedupeBarsByDate(all), nil
}

func (c *BinanceClient) fetchKlines(ctx context.Context, symbol string, start, end time.Time) ([]market.DailyBar, error) {
	u, err := url.Parse(c.baseURL + binanceKlinesPath)
	if err != nil {
		return nil, fmt.Errorf("binance: parse url: %w", err)
	}
	q := u.Query()
	q.Set("symbol", symbol)
	q.Set("interval", binanceInterval1Day)
	q.Set("startTime", strconv.FormatInt(start.UnixMilli(), 10))
	q.Set("endTime", strconv.FormatInt(end.UnixMilli(), 10))
	q.Set("limit", strconv.Itoa(binanceMaxKlines))
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("binance: build request: %w", err)
	}

	c.logger.Debug("binance request", "url", u.Redacted())

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("binance: http: %w", err)
	}
	defer func() {
		if closeErr := res.Body.Close(); closeErr != nil {
			c.logger.Warn("binance: close body", "error", closeErr)
		}
	}()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("binance: read body: %w", err)
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("binance: status %d: %s", res.StatusCode, truncate(string(body), 200))
	}

	var raw [][]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("binance: decode klines: %w", err)
	}
	return parseBinanceKlines(raw)
}

func parseBinanceKlines(raw [][]json.RawMessage) ([]market.DailyBar, error) {
	bars := make([]market.DailyBar, 0, len(raw))
	for _, row := range raw {
		if len(row) < 5 {
			return nil, fmt.Errorf("binance: unexpected kline row length %d", len(row))
		}
		openMS, err := parseJSONInt64(row[0])
		if err != nil {
			return nil, err
		}
		low, err := parseJSONFloat(row[3])
		if err != nil {
			return nil, err
		}
		close, err := parseJSONFloat(row[4])
		if err != nil {
			return nil, err
		}
		bars = append(bars, market.DailyBar{
			Date:  time.UnixMilli(openMS).UTC(),
			Low:   low,
			Close: close,
		})
	}
	return bars, nil
}

func parseJSONInt64(raw json.RawMessage) (int64, error) {
	var v int64
	if err := json.Unmarshal(raw, &v); err == nil {
		return v, nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return 0, fmt.Errorf("binance: parse int: %w", err)
	}
	return strconv.ParseInt(s, 10, 64)
}

func parseJSONFloat(raw json.RawMessage) (float64, error) {
	var v float64
	if err := json.Unmarshal(raw, &v); err == nil {
		return v, nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return 0, fmt.Errorf("binance: parse float: %w", err)
	}
	return strconv.ParseFloat(s, 64)
}

func dedupeBarsByDate(bars []market.DailyBar) []market.DailyBar {
	if len(bars) == 0 {
		return bars
	}
	out := make([]market.DailyBar, 0, len(bars))
	var last time.Time
	for _, b := range bars {
		day := b.Date.Truncate(24 * time.Hour)
		if !day.After(last) {
			continue
		}
		b.Date = day
		out = append(out, b)
		last = day
	}
	return out
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
