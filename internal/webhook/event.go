// Package webhook receives Kirim.email's delivery events (delivered, bounced,
// opened, clicked, …) and records them against the job that produced the email.
package webhook

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
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

// OccurredAt is the provider's own timestamp for the event. It is the third part
// of the idempotency key, so a webhook retry has to reproduce it exactly; when the
// payload carries none, fall back to now and accept that a retry may duplicate.
func (e Event) OccurredAt(fallback time.Time) time.Time {
	var detail struct {
		Timestamp int64 `json:"timestamp"`
	}
	if e.EventDetail == "" {
		return fallback
	}
	if err := json.Unmarshal([]byte(e.EventDetail), &detail); err != nil || detail.Timestamp <= 0 {
		return fallback
	}
	return time.Unix(detail.Timestamp, 0).In(fallback.Location())
}

// VerifySignature checks the payload's `signature` field.
//
// Kirim.email computes hash('sha256', $apiSecret . $messageGuid), and $apiSecret is
// the same API token we send email with (KIRIM_EMAIL_API_TOKEN) — confirmed against
// a real test payload from the dashboard, not assumed. Note what this does NOT
// cover: the event type,
// the recipient and the subject are unsigned, so a valid signature proves only
// that the sender knows the secret and the message guid. Treat the body as
// identifying, not as authoritative — anything acted on downstream should be
// cross-checked against our own rows.
func VerifySignature(secret, messageGUID, signature string) bool {
	if secret == "" {
		return false
	}
	sum := sha256.Sum256([]byte(secret + messageGUID))
	expected := hex.EncodeToString(sum[:])
	return subtle.ConstantTimeCompare([]byte(expected), []byte(strings.ToLower(strings.TrimSpace(signature)))) == 1
}
