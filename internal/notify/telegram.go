package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const telegramSendMessagePath = "/bot%s/sendMessage"

// TelegramClient sends messages via the Telegram Bot API.
type TelegramClient struct {
	token      string
	baseURL    string
	httpClient *http.Client
	logger     *slog.Logger
}

// TelegramOption configures TelegramClient.
type TelegramOption func(*TelegramClient)

// WithTelegramBaseURL overrides the API base (useful in tests).
func WithTelegramBaseURL(base string) TelegramOption {
	return func(c *TelegramClient) {
		c.baseURL = base
	}
}

// WithTelegramHTTPClient sets a custom HTTP client.
func WithTelegramHTTPClient(client *http.Client) TelegramOption {
	return func(c *TelegramClient) {
		c.httpClient = client
	}
}

// WithTelegramLogger sets the logger.
func WithTelegramLogger(logger *slog.Logger) TelegramOption {
	return func(c *TelegramClient) {
		c.logger = logger
	}
}

// NewTelegramClient creates a Telegram notifier.
func NewTelegramClient(token string, opts ...TelegramOption) (*TelegramClient, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, fmt.Errorf("notify: telegram token is required")
	}
	c := &TelegramClient{
		token:      token,
		baseURL:    "https://api.telegram.org",
		httpClient: &http.Client{Timeout: 30 * time.Second},
		logger:     slog.Default(),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c, nil
}

type sendMessageRequest struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode"`
}

// SendHTML delivers the same HTML message to every chat ID.
func (c *TelegramClient) SendHTML(ctx context.Context, chatIDs []string, text string) error {
	if len(chatIDs) == 0 {
		return fmt.Errorf("notify: no chat ids")
	}
	var errs []error
	for _, chatID := range chatIDs {
		if err := c.sendOne(ctx, chatID, text); err != nil {
			errs = append(errs, fmt.Errorf("chat %s: %w", chatID, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("notify: %w", errors.Join(errs...))
	}
	return nil
}

func (c *TelegramClient) sendOne(ctx context.Context, chatID, text string) error {
	path := fmt.Sprintf(telegramSendMessagePath, c.token)
	u, err := url.Parse(c.baseURL + path)
	if err != nil {
		return fmt.Errorf("notify: parse url: %w", err)
	}

	body, err := json.Marshal(sendMessageRequest{
		ChatID:    chatID,
		Text:      text,
		ParseMode: "HTML",
	})
	if err != nil {
		return fmt.Errorf("notify: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("notify: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	c.logger.Debug("telegram send", "chat_id", chatID)

	res, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("notify: http: %w", err)
	}
	defer func() {
		if closeErr := res.Body.Close(); closeErr != nil {
			c.logger.Warn("telegram: close body", "error", closeErr)
		}
	}()

	respBody, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("notify: read body: %w", err)
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("notify: status %d: %s", res.StatusCode, truncate(string(respBody), 200))
	}
	return nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
