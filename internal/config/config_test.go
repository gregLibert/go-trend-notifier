package config_test

import (
	"os"
	"strings"
	"testing"

	"github.com/gregLibert/go-trend-notifier/internal/config"
)

func setRequiredEnv(t *testing.T) {
	t.Helper()
	t.Setenv("TELEGRAM_TOKEN", "token")
	t.Setenv("TELEGRAM_CHAT_IDS", "1, 2")
	t.Setenv("BINANCE_ASSETS", "BTCUSDT, ETHUSDT")
	t.Setenv("YAHOO_ASSETS", "LQQ.PA,CL2.PA")
}

func TestLoad(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("CACHE_DIR", t.TempDir())

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.TelegramToken != "token" {
		t.Fatalf("token = %q", cfg.TelegramToken)
	}
	if len(cfg.TelegramChatIDs) != 2 {
		t.Fatalf("chat ids = %v", cfg.TelegramChatIDs)
	}
	if len(cfg.BinanceSymbols) != 2 {
		t.Fatalf("binance symbols = %v", cfg.BinanceSymbols)
	}
	if len(cfg.YahooSymbols) != 2 {
		t.Fatalf("yahoo symbols = %v", cfg.YahooSymbols)
	}
	if len(cfg.Assets) != 4 {
		t.Fatalf("assets = %d, want 4", len(cfg.Assets))
	}
}

func TestLoad_MissingToken(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("TELEGRAM_TOKEN", "")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLoad_NoAssets(t *testing.T) {
	t.Setenv("TELEGRAM_TOKEN", "token")
	t.Setenv("TELEGRAM_CHAT_IDS", "1")
	t.Setenv("BINANCE_ASSETS", "")
	t.Setenv("YAHOO_ASSETS", "  ,  ")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLoad_BinanceOnly(t *testing.T) {
	t.Setenv("TELEGRAM_TOKEN", "token")
	t.Setenv("TELEGRAM_CHAT_IDS", "1")
	t.Setenv("BINANCE_ASSETS", "BTCUSDT")
	t.Setenv("YAHOO_ASSETS", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Assets) != 1 || cfg.Assets[0].Provider != "binance" {
		t.Fatalf("assets = %+v", cfg.Assets)
	}
}

func TestParseCommaSeparated(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		want []string
	}{
		{name: "empty", raw: "", want: nil},
		{name: "whitespace only", raw: "  , , ", want: nil},
		{name: "single", raw: "BTCUSDT", want: []string{"BTCUSDT"}},
		{name: "trimmed list", raw: " BTCUSDT , ETHUSDT ", want: []string{"BTCUSDT", "ETHUSDT"}},
		{name: "skips empties", raw: "A,,B,", want: []string{"A", "B"}},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := config.ParseCommaSeparated(tt.raw)
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("got %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestBuildAssets(t *testing.T) {
	t.Parallel()

	_, err := config.BuildAssets(nil, nil)
	if err == nil {
		t.Fatal("expected error for empty asset lists")
	}

	assets, err := config.BuildAssets([]string{"BTCUSDT"}, []string{"LQQ.PA"})
	if err != nil {
		t.Fatalf("BuildAssets: %v", err)
	}
	if len(assets) != 2 {
		t.Fatalf("assets = %d, want 2", len(assets))
	}
	if assets[0].Provider != "binance" || assets[1].Provider != "yahoo" {
		t.Fatalf("providers = %s, %s", assets[0].Provider, assets[1].Provider)
	}
}

func TestLoadDotenv_MissingFileDoesNotPanic(t *testing.T) {
	t.Chdir(t.TempDir())
	config.LoadDotenv(nil)
}

func TestLoadDotenv_LoadsFile(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	content := strings.Join([]string{
		"TELEGRAM_TOKEN=from-dotenv",
		"TELEGRAM_CHAT_IDS=99",
		"BINANCE_ASSETS=BTCUSDT",
		"YAHOO_ASSETS=",
	}, "\n")
	if err := os.WriteFile(".env", []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	config.LoadDotenv(nil)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.TelegramToken != "from-dotenv" {
		t.Fatalf("token = %q, want from-dotenv", cfg.TelegramToken)
	}
}
