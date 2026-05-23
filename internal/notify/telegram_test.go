package notify_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gregLibert/go-trend-notifier/internal/notify"
)

func TestTelegramClient_SendHTML(t *testing.T) {
	t.Parallel()

	var gotBody map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	client, err := notify.NewTelegramClient("test-token", notify.WithTelegramBaseURL(srv.URL), notify.WithTelegramHTTPClient(srv.Client()))
	if err != nil {
		t.Fatalf("NewTelegramClient: %v", err)
	}

	text := "<b>alert</b>"
	if err := client.SendHTML(context.Background(), []string{"123"}, text); err != nil {
		t.Fatalf("SendHTML: %v", err)
	}
	if gotBody["chat_id"] != "123" {
		t.Fatalf("chat_id = %q", gotBody["chat_id"])
	}
	if gotBody["text"] != text {
		t.Fatalf("text = %q", gotBody["text"])
	}
	if gotBody["parse_mode"] != "HTML" {
		t.Fatalf("parse_mode = %q", gotBody["parse_mode"])
	}
}
