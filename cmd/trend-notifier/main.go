package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gregLibert/go-trend-notifier/internal/app"
	"github.com/gregLibert/go-trend-notifier/internal/config"
	"github.com/gregLibert/go-trend-notifier/internal/notify"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	config.LoadDotenv(logger)

	cfg, err := config.Load()
	if err != nil {
		logger.Error("configuration error", "error", err)
		os.Exit(1)
	}

	providers, err := app.BuildProviders(cfg, logger)
	if err != nil {
		logger.Error("provider setup failed", "error", err)
		os.Exit(1)
	}

	telegram, err := notify.NewTelegramClient(cfg.TelegramToken, notify.WithTelegramLogger(logger))
	if err != nil {
		logger.Error("telegram setup failed", "error", err)
		os.Exit(1)
	}

	runner, err := app.NewRunner(cfg, providers, telegram, logger)
	if err != nil {
		logger.Error("runner setup failed", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	runCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	if err := runner.Run(runCtx); err != nil {
		logger.Error("run failed", "error", err)
		os.Exit(1)
	}
}
