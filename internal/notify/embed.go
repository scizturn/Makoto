package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Discord's own limits. Exceed any of them and the webhook is rejected outright
// with a 400, so they are enforced here rather than discovered in production.
const (
	embedFieldLimit      = 25
	embedFieldNameLimit  = 256
	embedFieldValueLimit = 1024
	embedTitleLimit      = 256
)

// Colours for the left-hand bar. A human should be able to tell a bounce from a
// delivery without reading a word.
const (
	ColorSuccess = 0x2ECC71 // green  — delivered
	ColorInfo    = 0x3498DB // blue   — opened, clicked
	ColorNeutral = 0x95A5A6 // grey   — queued, send, deferred
	ColorWarning = 0xE67E22 // orange — unsubscribed, temporary failure
	ColorDanger  = 0xE74C3C // red    — bounced, permanent fail
)

// Embed is one Discord embed: a titled card with a coloured bar and a grid of
// fields, the shape the rest of the Kyou systems post in.
type Embed struct {
	Title       string
	Description string
	Color       int
	Fields      []Field
	Footer      string
	Timestamp   time.Time
}

type Field struct {
	Name   string
	Value  string
	Inline bool
}

// LogEmbed posts an embed. Like Log, it is a no-op when the webhook is not
// configured — a missing Discord URL must never fail the work that reports to it.
func (l DiscordLogger) LogEmbed(ctx context.Context, embed Embed) error {
	if !l.Enabled || l.WebhookURL == "" {
		return nil
	}

	payload := map[string]any{"embeds": []any{embed.toDiscord()}}

	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(payload); err != nil {
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
		return fmt.Errorf("discord embed webhook failed: status=%d", resp.StatusCode)
	}
	return nil
}

func (e Embed) toDiscord() map[string]any {
	payload := map[string]any{
		"title": truncateRunes(e.Title, embedTitleLimit),
		"color": e.Color,
	}
	if e.Description != "" {
		payload["description"] = truncateRunes(e.Description, 4096)
	}
	if e.Footer != "" {
		payload["footer"] = map[string]any{"text": truncateRunes(e.Footer, 2048)}
	}
	if !e.Timestamp.IsZero() {
		payload["timestamp"] = e.Timestamp.UTC().Format(time.RFC3339)
	}

	fields := make([]any, 0, len(e.Fields))
	for i, field := range e.Fields {
		if i == embedFieldLimit {
			break
		}
		// Discord rejects an embed outright if any field value is empty, so a field
		// with nothing to say is dropped rather than allowed to sink the whole post.
		if field.Value == "" {
			continue
		}
		fields = append(fields, map[string]any{
			"name":   truncateRunes(field.Name, embedFieldNameLimit),
			"value":  truncateRunes(field.Value, embedFieldValueLimit),
			"inline": field.Inline,
		})
	}
	if len(fields) > 0 {
		payload["fields"] = fields
	}
	return payload
}

func truncateRunes(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	if limit <= 1 {
		return string(runes[:limit])
	}
	return string(runes[:limit-1]) + "…"
}
