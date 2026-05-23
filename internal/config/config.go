package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	envTelegramToken   = "TELEGRAM_TOKEN"
	envTelegramChatIDs = "TELEGRAM_CHAT_IDS"
	envCacheDir        = "CACHE_DIR"
	envBinanceAssets   = "BINANCE_ASSETS"
	envYahooAssets     = "YAHOO_ASSETS"

	providerBinance = "binance"
	providerYahoo   = "yahoo"
)

// Asset defines a monitored symbol and its data provider key.
type Asset struct {
	Symbol   string
	Provider string
}

// Config holds runtime settings loaded from the environment.
type Config struct {
	TelegramToken   string
	TelegramChatIDs []string
	CacheDir        string
	CacheTTL        time.Duration
	BinanceSymbols  []string
	YahooSymbols    []string
	Assets          []Asset
}

// Load reads configuration from environment variables.
// Call LoadDotenv before Load when you want to merge a local .env file.
func Load() (Config, error) {
	token := strings.TrimSpace(os.Getenv(envTelegramToken))
	if token == "" {
		return Config{}, fmt.Errorf("config: %s is required", envTelegramToken)
	}

	chatRaw := strings.TrimSpace(os.Getenv(envTelegramChatIDs))
	if chatRaw == "" {
		return Config{}, fmt.Errorf("config: %s is required", envTelegramChatIDs)
	}
	chatIDs := ParseCommaSeparated(chatRaw)
	if len(chatIDs) == 0 {
		return Config{}, errors.New("config: no telegram chat ids parsed")
	}

	binance := ParseCommaSeparated(os.Getenv(envBinanceAssets))
	yahoo := ParseCommaSeparated(os.Getenv(envYahooAssets))
	assets, err := BuildAssets(binance, yahoo)
	if err != nil {
		return Config{}, err
	}

	cacheDir := strings.TrimSpace(os.Getenv(envCacheDir))
	if cacheDir == "" {
		cacheDir = ".cache"
	}

	return Config{
		TelegramToken:   token,
		TelegramChatIDs: chatIDs,
		CacheDir:        cacheDir,
		CacheTTL:        24 * time.Hour,
		BinanceSymbols:  binance,
		YahooSymbols:    yahoo,
		Assets:          assets,
	}, nil
}

// BuildAssets maps symbol lists to provider-tagged assets.
func BuildAssets(binance, yahoo []string) ([]Asset, error) {
	if len(binance) == 0 && len(yahoo) == 0 {
		return nil, fmt.Errorf("config: at least one symbol is required in %s or %s", envBinanceAssets, envYahooAssets)
	}

	out := make([]Asset, 0, len(binance)+len(yahoo))
	for _, symbol := range binance {
		out = append(out, Asset{Symbol: symbol, Provider: providerBinance})
	}
	for _, symbol := range yahoo {
		out = append(out, Asset{Symbol: symbol, Provider: providerYahoo})
	}
	return out, nil
}

// ParseCommaSeparated splits a comma-separated list, trims whitespace, and drops empty entries.
func ParseCommaSeparated(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
