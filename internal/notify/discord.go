package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type DiscordLogger struct {
	WebhookURL string
	Enabled    bool
	HTTPClient *http.Client
}

func (l DiscordLogger) Log(ctx context.Context, content string) error {
	if !l.Enabled || l.WebhookURL == "" {
		return nil
	}

	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(map[string]string{"content": content}); err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, l.WebhookURL, &body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := l.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("discord webhook failed: status=%d", resp.StatusCode)
	}
	return nil
}
