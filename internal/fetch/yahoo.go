package fetch

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/gregLibert/go-trend-notifier/internal/market"
)

const yahooChartPath = "/v8/finance/chart/"

// YahooClient fetches daily history from Yahoo Finance chart API.
type YahooClient struct {
	baseURL    string
	httpClient *http.Client
	logger     *slog.Logger
}

// YahooOption configures YahooClient.
type YahooOption func(*YahooClient)

// WithYahooBaseURL overrides the API base URL (useful in tests).
func WithYahooBaseURL(base string) YahooOption {
	return func(c *YahooClient) {
		c.baseURL = base
	}
}

// WithYahooHTTPClient sets a custom HTTP client.
func WithYahooHTTPClient(client *http.Client) YahooOption {
	return func(c *YahooClient) {
		c.httpClient = client
	}
}

// WithYahooLogger sets the logger.
func WithYahooLogger(logger *slog.Logger) YahooOption {
	return func(c *YahooClient) {
		c.logger = logger
	}
}

// NewYahooClient creates a Yahoo Finance chart client.
func NewYahooClient(opts ...YahooOption) *YahooClient {
	c := &YahooClient{
		baseURL:    "https://query1.finance.yahoo.com",
		httpClient: &http.Client{Timeout: 30 * time.Second},
		logger:     slog.Default(),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *YahooClient) Name() string {
	return "yahoo"
}

// FetchDailyBars downloads two years of daily candles.
func (c *YahooClient) FetchDailyBars(ctx context.Context, symbol string) ([]market.DailyBar, error) {
	u, err := url.Parse(c.baseURL + yahooChartPath + url.PathEscape(symbol))
	if err != nil {
		return nil, fmt.Errorf("yahoo: parse url: %w", err)
	}
	q := u.Query()
	q.Set("range", "2y")
	q.Set("interval", "1d")
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("yahoo: build request: %w", err)
	}
	req.Header.Set("User-Agent", "go-trend-notifier/1.0")

	c.logger.Debug("yahoo request", "url", u.Redacted())

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("yahoo: http: %w", err)
	}
	defer func() {
		if closeErr := res.Body.Close(); closeErr != nil {
			c.logger.Warn("yahoo: close body", "error", closeErr)
		}
	}()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("yahoo: read body: %w", err)
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("yahoo: status %d: %s", res.StatusCode, truncate(string(body), 200))
	}

	return parseYahooChart(body)
}

type yahooChartResponse struct {
	Chart struct {
		Result []struct {
			Timestamp  []int64 `json:"timestamp"`
			Indicators struct {
				Quote []struct {
					Close []float64 `json:"close"`
					Low   []float64 `json:"low"`
				} `json:"quote"`
			} `json:"indicators"`
		} `json:"result"`
		Error *struct {
			Description string `json:"description"`
		} `json:"error"`
	} `json:"chart"`
}

func parseYahooChart(body []byte) ([]market.DailyBar, error) {
	var payload yahooChartResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("yahoo: decode: %w", err)
	}
	if payload.Chart.Error != nil {
		return nil, fmt.Errorf("yahoo: api error: %s", payload.Chart.Error.Description)
	}
	if len(payload.Chart.Result) == 0 {
		return nil, fmt.Errorf("yahoo: empty chart result")
	}
	result := payload.Chart.Result[0]
	if len(result.Indicators.Quote) == 0 {
		return nil, fmt.Errorf("yahoo: missing quote indicators")
	}
	quote := result.Indicators.Quote[0]
	if len(result.Timestamp) == 0 {
		return nil, fmt.Errorf("yahoo: no timestamps")
	}

	bars := make([]market.DailyBar, 0, len(result.Timestamp))
	for i, ts := range result.Timestamp {
		close, okClose := valueAt(quote.Close, i)
		low, okLow := valueAt(quote.Low, i)
		if !okClose || !okLow {
			continue
		}
		bars = append(bars, market.DailyBar{
			Date:  time.Unix(ts, 0).UTC(),
			Close: close,
			Low:   low,
		})
	}
	if len(bars) == 0 {
		return nil, fmt.Errorf("yahoo: no valid daily bars")
	}
	return bars, nil
}

func valueAt(series []float64, i int) (float64, bool) {
	if i >= len(series) {
		return 0, false
	}
	v := series[i]
	if v != v { // NaN
		return 0, false
	}
	return v, true
}
