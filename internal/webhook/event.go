// Package webhook receives Kirim.email's delivery events (delivered, bounced,
// opened, clicked, …) and records them against the job that produced the email.
package webhook

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Event is one Kirim.email webhook payload, as the provider posts it.
type Event struct {
	MessageGUID string `json:"message_guid"`
	Type        string `json:"type"`
	Sender      string `json:"sender"`
	Recipient   string `json:"recipient"`
	EventType   string `json:"event_type"`
	Event       string `json:"event"`
	Subject     string `json:"subject"`
	Status      string `json:"status"`
	Tags        string `json:"tags"`
	Signature   string `json:"signature"`

	// EventDetail is a JSON document delivered as a *string*, e.g.
	// "{\"timestamp\":1783932326}" — not a nested object.
	EventDetail string `json:"event_detail"`
}

// ParsePayload reads a webhook body in whichever shape Kirim.email chose to send it.
//
// The dashboard displays the payload as JSON, but that is the payload it *means* to
// send, not necessarily the bytes on the wire — a JSON post is rejected as malformed
// unless it really is JSON. So: try JSON, then form-encoded (both the flat form and
// the "one field holding the JSON" form providers often use). Whatever arrives, the
// signature still has to check out, so being liberal here costs no safety.
func ParsePayload(body []byte) (Event, error) {
	var event Event
	jsonErr := json.Unmarshal(body, &event)
	if jsonErr == nil {
		return event, nil
	}

	form, formErr := url.ParseQuery(string(body))
	if formErr != nil {
		return Event{}, jsonErr
	}

	// A single field carrying the whole JSON document (payload=…, data=…, body=…).
	for _, field := range []string{"payload", "data", "body", "event", "json"} {
		nested := form.Get(field)
		if nested == "" || !strings.HasPrefix(strings.TrimSpace(nested), "{") {
			continue
		}
		var wrapped Event
		if err := json.Unmarshal([]byte(nested), &wrapped); err == nil && wrapped.MessageGUID != "" {
			return wrapped, nil
		}
	}

	// A flat form: message_guid=…&event_type=…&signature=…
	flat := Event{
		MessageGUID: form.Get("message_guid"),
		Type:        form.Get("type"),
		Sender:      form.Get("sender"),
		Recipient:   form.Get("recipient"),
		EventType:   form.Get("event_type"),
		Event:       form.Get("event"),
		Subject:     form.Get("subject"),
		Status:      form.Get("status"),
		Tags:        form.Get("tags"),
		Signature:   form.Get("signature"),
		EventDetail: form.Get("event_detail"),
	}
	if flat.MessageGUID != "" && flat.Signature != "" {
		return flat, nil
	}

	// Neither shape produced anything usable. Return the JSON error — it describes the
	// shape we expect, and the caller logs the raw body next to it.
	return Event{}, jsonErr
}

// Kind normalises the event name. Kirim.email sends `event_type` and `event` with
// the same value in the payloads we have seen, but only one of them is documented,
// so the others are accepted as fallbacks rather than trusted to be present.
func (e Event) Kind() string {
	for _, candidate := range []string{e.EventType, e.Event, e.Status} {
		if kind := strings.ToLower(strings.TrimSpace(candidate)); kind != "" {
			return kind
		}
	}
	return ""
}

// JobID is the job this email came from. Makoto stamps it into X-Tags at send time
// and Kirim.email hands it back in `tags` — the send API returns no message id, so
// this is the only key that ties an event to a row in email_delivery_logs.
//
// Kirim.email treats tags as a comma-separated list, so take the first entry.
func (e Event) JobID() string {
	tag, _, _ := strings.Cut(e.Tags, ",")
	return strings.TrimSpace(tag)
}

// OccurredAt is the provider's own timestamp for the event.
//
// The key it hides under changes with the event: `timestamp` on a queued event,
// `delivered_at` on a delivered one, and presumably its own name on each of the
// rest. Rather than enumerate names we will keep discovering one production
// surprise at a time, take the first plausible unix timestamp in the document.
// If there is none, fall back to our own clock — a slightly late timestamp beats
// dropping the event.
func (e Event) OccurredAt(fallback time.Time) time.Time {
	if e.EventDetail == "" {
		return fallback
	}
	var detail map[string]any
	if err := json.Unmarshal([]byte(e.EventDetail), &detail); err != nil {
		return fallback
	}

	// Preferred names first, then anything *_at / timestamp-ish the payload offers.
	// The discovered names are sorted: map iteration is randomised in Go, and a
	// payload with two timestamps must not pick a different one on each event.
	discovered := make([]string, 0, len(detail))
	for key := range detail {
		if strings.HasSuffix(key, "_at") || strings.Contains(key, "time") {
			discovered = append(discovered, key)
		}
	}
	sort.Strings(discovered)

	keys := append([]string{"timestamp", "event_at"}, discovered...)
	for _, key := range keys {
		seconds, ok := unixSeconds(detail[key])
		if ok {
			return time.Unix(seconds, 0).In(fallback.Location())
		}
	}
	return fallback
}

// unixSeconds accepts the number as JSON gives it (float64) or as a string, and
// rejects anything that is not a plausible epoch — a zero, a millisecond value, or
// a "250 2.0.0 OK" smtp_response that happens to sit in the same document.
func unixSeconds(value any) (int64, bool) {
	var seconds int64
	switch typed := value.(type) {
	case float64:
		seconds = int64(typed)
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
		if err != nil {
			return 0, false
		}
		seconds = parsed
	default:
		return 0, false
	}

	// 2001-09-09 .. 2286-11-20, i.e. ten digits.
	if seconds < 1_000_000_000 || seconds > 9_999_999_999 {
		return 0, false
	}
	return seconds, true
}

// VerifySignature checks the payload's `signature` field.
//
// Kirim.email computes hash('sha256', $apiSecret . $messageGuid), and $apiSecret is
// the same API token we send email with (KIRIM_EMAIL_API_TOKEN) — confirmed against
// a real test payload from the dashboard, not assumed.
//
// Note what this does NOT cover: the event type, the recipient and the subject are
// unsigned, so a valid signature proves only that the sender knows the secret and
// the message guid. Treat the body as identifying, not as authoritative — anything
// acted on downstream should be cross-checked against our own rows.
func VerifySignature(secret, messageGUID, signature string) bool {
	if secret == "" {
		return false
	}
	sum := sha256.Sum256([]byte(secret + messageGUID))
	expected := hex.EncodeToString(sum[:])
	return subtle.ConstantTimeCompare([]byte(expected), []byte(strings.ToLower(strings.TrimSpace(signature)))) == 1
}
