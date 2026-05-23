package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/gregLibert/go-trend-notifier/internal/config"
	"github.com/gregLibert/go-trend-notifier/internal/fetch"
	"github.com/gregLibert/go-trend-notifier/internal/market"
	"github.com/gregLibert/go-trend-notifier/internal/notify"
)

// Runner orchestrates fetch, indicator computation, evaluation, and notification.
type Runner struct {
	cfg       config.Config
	providers map[string]fetch.Provider
	telegram  *notify.TelegramClient
	logger    *slog.Logger
	now       func() time.Time
}

// NewRunner wires dependencies for a monitoring run.
func NewRunner(cfg config.Config, providers map[string]fetch.Provider, telegram *notify.TelegramClient, logger *slog.Logger) (*Runner, error) {
	if len(providers) == 0 {
		return nil, fmt.Errorf("app: no providers configured")
	}
	if telegram == nil {
		return nil, fmt.Errorf("app: telegram client is required")
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Runner{
		cfg:       cfg,
		providers: providers,
		telegram:  telegram,
		logger:    logger,
		now:       time.Now,
	}, nil
}

// Run executes one full monitoring cycle.
func (r *Runner) Run(ctx context.Context) error {
	alerts, err := r.collectAlerts(ctx)
	if len(alerts) == 0 {
		if err != nil {
			return err
		}
		r.logger.Info("no alert conditions met")
		return nil
	}

	message := market.FormatAlertMessage(alerts, r.now())
	r.logger.Info("sending telegram alert", "assets", len(alerts))
	sendErr := r.telegram.SendHTML(ctx, r.cfg.TelegramChatIDs, message)
	if sendErr != nil {
		sendErr = fmt.Errorf("app: telegram: %w", sendErr)
	}
	if sendErr != nil {
		return errors.Join(err, sendErr)
	}
	if err != nil {
		r.logger.Warn("alert sent with partial fetch failures", "error", err)
		return err
	}
	r.logger.Info("alert sent successfully")
	return nil
}

func (r *Runner) collectAlerts(ctx context.Context) ([]market.AssetAlert, error) {
	if len(r.cfg.Assets) == 0 {
		return nil, nil
	}

	var (
		wg     sync.WaitGroup
		mu     sync.Mutex
		alerts []market.AssetAlert
		errs   []error
	)

	for _, asset := range r.cfg.Assets {
		wg.Add(1)
		go func(asset config.Asset) {
			defer wg.Done()
			r.collectOneAsset(ctx, asset, &mu, &alerts, &errs)
		}(asset)
	}

	wg.Wait()

	if len(errs) == 0 {
		return alerts, nil
	}
	return alerts, errors.Join(errs...)
}

func (r *Runner) collectOneAsset(ctx context.Context, asset config.Asset, mu *sync.Mutex, alerts *[]market.AssetAlert, errs *[]error) {
	alert, triggered, err := r.evaluateAsset(ctx, asset)
	if err != nil {
		r.logger.Error("asset evaluation failed", "symbol", asset.Symbol, "provider", asset.Provider, "error", err)
		mu.Lock()
		*errs = append(*errs, fmt.Errorf("%s (%s): %w", asset.Symbol, asset.Provider, err))
		mu.Unlock()
		return
	}
	if !triggered {
		return
	}
	mu.Lock()
	*alerts = append(*alerts, alert)
	mu.Unlock()
}

func (r *Runner) evaluateAsset(ctx context.Context, asset config.Asset) (market.AssetAlert, bool, error) {
	provider, ok := r.providers[asset.Provider]
	if !ok {
		return market.AssetAlert{}, false, fmt.Errorf("unknown provider %q", asset.Provider)
	}

	r.logger.Info("fetching bars", "symbol", asset.Symbol, "provider", asset.Provider)
	bars, err := provider.FetchDailyBars(ctx, asset.Symbol)
	if err != nil {
		return market.AssetAlert{}, false, fmt.Errorf("fetch daily bars: %w", err)
	}

	snapshot, err := market.ComputeSnapshot(asset.Symbol, asset.Provider, bars)
	if err != nil {
		return market.AssetAlert{}, false, fmt.Errorf("compute snapshot: %w", err)
	}

	conditions := market.EvaluateConditions(snapshot)
	if len(conditions) == 0 {
		r.logger.Info("ok", "symbol", asset.Symbol, "close", snapshot.Current)
		return market.AssetAlert{}, false, nil
	}

	r.logger.Warn("alert conditions met", "symbol", asset.Symbol, "conditions", len(conditions))
	return market.AssetAlert{Snapshot: snapshot, Conditions: conditions}, true, nil
}

// BuildProviders creates cached Binance and Yahoo providers from config.
func BuildProviders(cfg config.Config, logger *slog.Logger) (map[string]fetch.Provider, error) {
	binance := fetch.NewBinanceClient(fetch.WithBinanceLogger(logger))
	yahoo := fetch.NewYahooClient(fetch.WithYahooLogger(logger))

	binanceCached, err := fetch.NewCachedProvider(binance, cfg.CacheDir, cfg.CacheTTL, logger)
	if err != nil {
		return nil, fmt.Errorf("app: binance cache: %w", err)
	}
	yahooCached, err := fetch.NewCachedProvider(yahoo, cfg.CacheDir, cfg.CacheTTL, logger)
	if err != nil {
		return nil, fmt.Errorf("app: yahoo cache: %w", err)
	}

	return map[string]fetch.Provider{
		"binance": binanceCached,
		"yahoo":   yahooCached,
	}, nil
}
