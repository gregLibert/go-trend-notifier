package fetch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/gregLibert/go-trend-notifier/internal/market"
)

const defaultCacheTTL = 24 * time.Hour

// CachedProvider wraps a Provider with on-disk JSON caching.
type CachedProvider struct {
	inner  Provider
	dir    string
	ttl    time.Duration
	now    func() time.Time
	logger *slog.Logger
}

type cacheEntry struct {
	FetchedAt time.Time         `json:"fetched_at"`
	Bars      []market.DailyBar `json:"bars"`
}

// NewCachedProvider returns a disk-caching wrapper. cacheDir is created if missing.
func NewCachedProvider(inner Provider, cacheDir string, ttl time.Duration, logger *slog.Logger) (*CachedProvider, error) {
	if inner == nil {
		return nil, errors.New("fetch: inner provider is nil")
	}
	if cacheDir == "" {
		return nil, errors.New("fetch: cache directory is required")
	}
	if ttl <= 0 {
		ttl = defaultCacheTTL
	}
	if logger == nil {
		logger = slog.Default()
	}
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return nil, fmt.Errorf("fetch: create cache dir: %w", err)
	}
	return &CachedProvider{
		inner:  inner,
		dir:    cacheDir,
		ttl:    ttl,
		now:    time.Now,
		logger: logger,
	}, nil
}

func (c *CachedProvider) Name() string {
	return c.inner.Name()
}

// FetchDailyBars returns cached bars when fresh; otherwise refetches and updates the cache.
func (c *CachedProvider) FetchDailyBars(ctx context.Context, symbol string) ([]market.DailyBar, error) {
	path := c.cachePath(symbol)
	if bars, ok, err := c.loadFresh(path); err != nil {
		return nil, err
	} else if ok {
		c.logger.Info("cache hit", "provider", c.inner.Name(), "symbol", symbol, "path", path)
		return bars, nil
	}

	c.logger.Info("cache miss", "provider", c.inner.Name(), "symbol", symbol)
	bars, err := c.inner.FetchDailyBars(ctx, symbol)
	if err != nil {
		return nil, fmt.Errorf("fetch: %s %s: %w", c.inner.Name(), symbol, err)
	}
	if err := c.save(path, bars); err != nil {
		return nil, fmt.Errorf("fetch: save cache: %w", err)
	}
	return bars, nil
}

func (c *CachedProvider) cachePath(symbol string) string {
	filename := fmt.Sprintf("%s_%s.json", c.inner.Name(), symbol)
	return filepath.Join(c.dir, filename)
}

func (c *CachedProvider) loadFresh(path string) ([]market.DailyBar, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("fetch: read cache %s: %w", path, err)
	}

	var entry cacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		c.logger.Warn("invalid cache file, refetching", "path", path, "error", err)
		return nil, false, nil
	}
	if c.now().Sub(entry.FetchedAt) > c.ttl {
		return nil, false, nil
	}
	if len(entry.Bars) == 0 {
		return nil, false, nil
	}
	return entry.Bars, true, nil
}

func (c *CachedProvider) save(path string, bars []market.DailyBar) error {
	entry := cacheEntry{
		FetchedAt: c.now(),
		Bars:      bars,
	}
	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return fmt.Errorf("fetch: marshal cache: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("fetch: write cache temp: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("fetch: rename cache: %w", err)
	}
	c.logger.Info("cache updated", "path", path, "bars", len(bars))
	return nil
}
