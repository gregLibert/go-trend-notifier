# go-trend-notifier

A lightweight Go binary that monitors crypto/equity tickers and sends a Telegram alert if prices drop below their 50/200-day moving average or period lows.

## Quick Start
```
    cp .env.example .env
    # Edit .env with your tokens and assets
    go run ./cmd/trend-notifier
```

## Configuration

Configure via .env file or OS environment variables.

| Variable | Description | Example |
|---|---|---|
| TELEGRAM_TOKEN | Required. Bot API token | 1234:abcd |
| TELEGRAM_CHAT_IDS | Required. Target chats | -1001234567890 |
| BINANCE_ASSETS | Binance Spot symbols | BTCUSDT,ETHUSDT |
| YAHOO_ASSETS | Yahoo Finance symbols | LQQ.PA,CL2.PA |
| CACHE_DIR | Disk cache (24h TTL) | .cache |

*(Note: Provide at least one asset in either Binance or Yahoo).*

## Build & Deploy (ARM64 / Orange Pi)

Build a static binary for instance for an ARM device:
```
    CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o trend-notifier-arm64 ./cmd/trend-notifier
```
Execute the binary directly (ensure env vars are set):
```
    ./trend-notifier-arm64
```
> Pro-tip: Schedule this binary via cron (e.g., `0 18 * * 1-5 /opt/bot/trend-notifier-arm64`) to run daily after market close. Logs are output as JSON to stdout.

## Testing

Standard unit tests:
```
    go test -v -race ./...
```
Live integration tests (calls actual Binance/Yahoo APIs):
```
    go test -v -tags=integration ./internal/fetch
```