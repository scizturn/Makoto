package webhook

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/kyou-id/makoto/internal/mask"
)

// maxBodyBytes caps what we will read from a webhook post. The payloads are a few
// hundred bytes; anything larger is either a bug or an attempt to make us allocate.
const maxBodyBytes = 64 << 10

// Recorder persists one delivery event. It is an interface so the handler can be
// tested without MySQL.
type Recorder interface {
	RecordDeliveryEvent(ctx context.Context, event Record) error
}

// Notifier posts a line to Discord.
type Notifier interface {
	Log(ctx context.Context, content string) error
}

// Record is what we keep from an event, resolved to our own identifiers.
type Record struct {
	JobID       string
	MessageGUID string
	EventType   string
	ToEmail     string
	Subject     string
	Tags        string
	OccurredAt  time.Time
	Payload     string
}

// Handler is the HTTP endpoint Kirim.email posts to.
type Handler struct {
	Secret   string
	Recorder Recorder
	Discord  Notifier

	// NotifyEvents are the event kinds worth a Discord line. Deliveries and opens
	// arrive in the thousands and would drown the channel; bounces are the ones a
	// human should see.
	NotifyEvents map[string]bool

	Now func() time.Time
}

func (h Handler) now() time.Time {
	if h.Now != nil {
		return h.Now()
	}
	return time.Now()
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, maxBodyBytes))
	if err != nil {
		http.Error(w, "cannot read body", http.StatusBadRequest)
		return
	}

	event, err := ParsePayload(body)
	if err != nil {
		// Log what actually came down the wire. The dashboard shows the payload it
		// *means* to send, which is not the same thing as the bytes it sends, and
		// guessing at the difference is how you burn a deploy cycle per attempt.
		log.Printf("kirim webhook: unreadable payload: %v | content-type=%q | body=%.500q",
			err, r.Header.Get("Content-Type"), body)
		http.Error(w, "malformed payload", http.StatusBadRequest)
		return
	}

	// An unverifiable event is refused, not recorded: without the signature check
	// anyone who guesses the URL can write rows claiming our emails bounced.
	if !VerifySignature(h.Secret, event.MessageGUID, event.Signature) {
		log.Printf("kirim webhook: bad signature for message_guid=%s event=%s", event.MessageGUID, event.Kind())
		http.Error(w, "bad signature", http.StatusUnauthorized)
		return
	}

	kind := event.Kind()
	if kind == "" || event.MessageGUID == "" {
		http.Error(w, "missing event_type or message_guid", http.StatusBadRequest)
		return
	}

	record := Record{
		JobID:       event.JobID(),
		MessageGUID: event.MessageGUID,
		EventType:   kind,
		ToEmail:     event.Recipient,
		Subject:     event.Subject,
		Tags:        event.Tags,
		OccurredAt:  event.OccurredAt(h.now()),
		Payload:     string(body),
	}

	if h.Recorder != nil {
		// A 5xx makes Kirim.email retry, which is what we want when the DB is down:
		// the event is not lost, it comes back. The write is idempotent, so a retry
		// of an event we did record is harmless.
		if err := h.Recorder.RecordDeliveryEvent(r.Context(), record); err != nil {
			log.Printf("kirim webhook: record %s for job %q failed: %v", kind, record.JobID, err)
			http.Error(w, "cannot record event", http.StatusInternalServerError)
			return
		}
	}

	if record.JobID == "" {
		// The X-Tags stamp is the only thing tying an event to a job. Missing means
		// either an email sent before Makoto stamped them, or the stamp is not coming
		// back — and the second case makes every event from here on unattributable.
		log.Printf("kirim webhook: %s for %s has NO job tag (guid=%s tags=%q) — nothing to attach it to",
			kind, MaskEmail(record.ToEmail), record.MessageGUID, record.Tags)
	} else {
		log.Printf("kirim webhook: %s job=%s email=%s guid=%s", kind, record.JobID, MaskEmail(record.ToEmail), record.MessageGUID)
	}

	h.notify(r.Context(), record)

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"ok":true}`))
}

// notify posts the noteworthy events to Discord. It never fails the request — the
// event is already stored, and making Kirim.email retry because *Discord* is down
// would replay it forever.
func (h Handler) notify(ctx context.Context, record Record) {
	if h.Discord == nil || !h.NotifyEvents[record.EventType] {
		return
	}
	job := record.JobID
	if job == "" {
		job = "(untagged: " + record.MessageGUID + ")"
	}
	content := fmt.Sprintf("⚠️ [Kirim.email %s]\nJob: %s\nEmail: %s\nSubject: %s",
		strings.ToUpper(record.EventType), job, MaskEmail(record.ToEmail), record.Subject)
	if err := h.Discord.Log(ctx, content); err != nil {
		log.Printf("kirim webhook: discord notify failed: %v", err)
	}
}

// MaskEmail hides the local part before an address reaches a log line or Discord.
func MaskEmail(email string) string {
	return mask.Email(email)
}
