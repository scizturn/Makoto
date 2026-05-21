package notify

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDiscordLoggerPostsContent(t *testing.T) {
	var body map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	logger := DiscordLogger{WebhookURL: server.URL, Enabled: true, HTTPClient: server.Client()}
	err := logger.Log(context.Background(), "birthday email sent")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if body["content"] != "birthday email sent" {
		t.Fatalf("expected discord content, got %#v", body)
	}
}
