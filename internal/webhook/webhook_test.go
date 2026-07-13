package webhook

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

const testSecret = "s3cr3t"

func signatureFor(guid string) string {
	sum := sha256.Sum256([]byte(testSecret + guid))
	return hex.EncodeToString(sum[:])
}

// The payload Kirim.email's "Test Webhook" button posts, shape-for-shape.
func testPayload(guid, eventType, tags string) string {
	payload := map[string]string{
		"message_guid": guid,
		"type":         "email",
		"sender":       "nandayo@kyou.id",
		"recipient":    "someone@example.com",
		"event_type":   eventType,
		"event":        eventType,
		"subject":      "Selamat ulang tahun",
		"status":       eventType,
		"tags":         tags,
		"signature":    signatureFor(guid),
		"event_detail": `{"timestamp":1783932326}`,
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}
	return string(encoded)
}

type fakeRecorder struct {
	records []Record
	err     error
}

func (f *fakeRecorder) RecordDeliveryEvent(_ context.Context, record Record) error {
	if f.err != nil {
		return f.err
	}
	f.records = append(f.records, record)
	return nil
}

type fakeNotifier struct{ messages []string }

func (f *fakeNotifier) Log(_ context.Context, content string) error {
	f.messages = append(f.messages, content)
	return nil
}

func post(t *testing.T, handler Handler, body string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/webhooks/kirim", strings.NewReader(body))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func TestHandlerRecordsEventAgainstTheJob(t *testing.T) {
	recorder := &fakeRecorder{}
	handler := Handler{Secret: testSecret, Recorder: recorder}

	response := post(t, handler, testPayload("guid-1", "delivered", "birthday-2026-07-13-user-42"))

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body)
	}
	if len(recorder.records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(recorder.records))
	}
	got := recorder.records[0]
	if got.JobID != "birthday-2026-07-13-user-42" {
		t.Fatalf("expected the job id from the tags, got %q", got.JobID)
	}
	if got.EventType != "delivered" {
		t.Fatalf("expected delivered, got %q", got.EventType)
	}
	// The provider's own timestamp, not our clock — a retry must reproduce it.
	if want := time.Unix(1783932326, 0); !got.OccurredAt.Equal(want) {
		t.Fatalf("expected occurred_at %s, got %s", want, got.OccurredAt)
	}
}

// Without this check, anyone who guesses the URL can write rows claiming our
// emails bounced.
func TestHandlerRejectsABadSignature(t *testing.T) {
	recorder := &fakeRecorder{}
	handler := Handler{Secret: testSecret, Recorder: recorder}

	body := strings.Replace(
		testPayload("guid-1", "bounced", "birthday-1"),
		signatureFor("guid-1"),
		strings.Repeat("a", 64),
		1,
	)
	response := post(t, handler, body)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", response.Code)
	}
	if len(recorder.records) != 0 {
		t.Fatal("a payload with a bad signature must not be recorded")
	}
}

// A signature valid for one message must not be reusable for another.
func TestSignatureIsBoundToTheMessageGUID(t *testing.T) {
	if !VerifySignature(testSecret, "guid-1", signatureFor("guid-1")) {
		t.Fatal("expected the matching signature to verify")
	}
	if VerifySignature(testSecret, "guid-2", signatureFor("guid-1")) {
		t.Fatal("a signature for guid-1 must not verify guid-2")
	}
	if VerifySignature("", "guid-1", signatureFor("guid-1")) {
		t.Fatal("an empty secret must never verify")
	}
}

// Kirim.email retries until it gets a 2xx. A DB failure therefore has to answer 5xx
// so the event comes back, rather than 200 which would drop it for good.
func TestHandlerAsksForARetryWhenTheDatabaseIsDown(t *testing.T) {
	handler := Handler{Secret: testSecret, Recorder: &fakeRecorder{err: context.DeadlineExceeded}}

	response := post(t, handler, testPayload("guid-1", "delivered", "birthday-1"))

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 so Kirim.email retries, got %d", response.Code)
	}
}

func TestHandlerAnnouncesOnlyTheNoteworthyEvents(t *testing.T) {
	discord := &fakeNotifier{}
	handler := Handler{
		Secret:       testSecret,
		Recorder:     &fakeRecorder{},
		Discord:      discord,
		NotifyEvents: map[string]bool{"bounced": true},
	}

	post(t, handler, testPayload("guid-1", "delivered", "birthday-1"))
	post(t, handler, testPayload("guid-2", "opened", "birthday-1"))
	if len(discord.messages) != 0 {
		t.Fatalf("deliveries and opens would drown the channel, got %v", discord.messages)
	}

	post(t, handler, testPayload("guid-3", "bounced", "birthday-2026-07-13-user-42"))
	if len(discord.messages) != 1 {
		t.Fatalf("expected a bounce to be announced, got %d messages", len(discord.messages))
	}
	message := discord.messages[0]
	if !strings.Contains(message, "birthday-2026-07-13-user-42") {
		t.Fatalf("expected the job id in the message, got %q", message)
	}
	if strings.Contains(message, "someone@example.com") {
		t.Fatalf("the recipient must be masked, got %q", message)
	}
}

// Kirim.email treats tags as a comma-separated list, so the job id is the first
// entry — not the whole field.
func TestJobIDTakesTheFirstTag(t *testing.T) {
	event := Event{Tags: "birthday-2026-07-13-user-42,promo"}
	if got := event.JobID(); got != "birthday-2026-07-13-user-42" {
		t.Fatalf("got %q", got)
	}
	if got := (Event{}).JobID(); got != "" {
		t.Fatalf("expected an empty job id for an untagged email, got %q", got)
	}
}

func TestOccurredAtFallsBackWhenTheTimestampIsMissing(t *testing.T) {
	fallback := time.Date(2026, 7, 13, 9, 0, 0, 0, time.UTC)
	if got := (Event{}).OccurredAt(fallback); !got.Equal(fallback) {
		t.Fatalf("expected the fallback, got %s", got)
	}
	if got := (Event{EventDetail: "not json"}).OccurredAt(fallback); !got.Equal(fallback) {
		t.Fatalf("expected the fallback on malformed detail, got %s", got)
	}
}

func TestHandlerRejectsNonPost(t *testing.T) {
	handler := Handler{Secret: testSecret, Recorder: &fakeRecorder{}}
	request := httptest.NewRequest(http.MethodGet, "/webhooks/kirim", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", response.Code)
	}
}

// The exact "Delivered" payload the dashboard's Test button sends. Its timestamp
// hides under `delivered_at`, not `timestamp` — and it shares the document with an
// smtp_response string that must not be mistaken for one.
const deliveredPayload = `{
  "message_guid": "test-6a54b777ab1e0",
  "type": "email",
  "sender": "test-sender@example.com",
  "sender_domain": "example.com",
  "sender_ip": "192.168.1.100",
  "recipient": "test-recipient@example.com",
  "recipient_domain": "example.com",
  "recipient_ip": "192.168.1.101",
  "event_type": "delivered",
  "event": "delivered",
  "subject": "Test Webhook Email Subject",
  "status": "delivered",
  "tags": "test,webhook",
  "signature": "%s",
  "event_detail": "{\"smtp_response\":\"250 2.0.0 OK: queued as 12345ABC\",\"delivered_at\":1783936887}"
}`

func TestHandlerAcceptsTheDashboardDeliveredPayload(t *testing.T) {
	recorder := &fakeRecorder{}
	handler := Handler{Secret: testSecret, Recorder: recorder}

	body := fmt.Sprintf(deliveredPayload, signatureFor("test-6a54b777ab1e0"))
	response := post(t, handler, body)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body)
	}
	got := recorder.records[0]
	if want := time.Unix(1783936887, 0); !got.OccurredAt.Equal(want) {
		t.Fatalf("expected the delivered_at timestamp %s, got %s", want, got.OccurredAt)
	}
}

// Kirim.email's own test post came back 400 "malformed payload" in production, so
// the body on the wire is not always the JSON the dashboard displays. Accept the
// form-encoded shapes too — the signature still has to check out either way.
func TestParsePayloadAcceptsFormEncodedBodies(t *testing.T) {
	guid := "guid-form-1"

	flat := url.Values{
		"message_guid": {guid},
		"event_type":   {"bounced"},
		"recipient":    {"someone@example.com"},
		"tags":         {"birthday-2026-07-13-user-42"},
		"signature":    {signatureFor(guid)},
	}
	event, err := ParsePayload([]byte(flat.Encode()))
	if err != nil {
		t.Fatalf("flat form: %v", err)
	}
	if event.Kind() != "bounced" || event.JobID() != "birthday-2026-07-13-user-42" {
		t.Fatalf("flat form decoded wrong: %#v", event)
	}

	nested := url.Values{"payload": {fmt.Sprintf(`{"message_guid":%q,"event_type":"opened","signature":%q}`, guid, signatureFor(guid))}}
	event, err = ParsePayload([]byte(nested.Encode()))
	if err != nil {
		t.Fatalf("nested form: %v", err)
	}
	if event.MessageGUID != guid || event.Kind() != "opened" {
		t.Fatalf("nested form decoded wrong: %#v", event)
	}
}

func TestParsePayloadRejectsGarbage(t *testing.T) {
	if _, err := ParsePayload([]byte("this is not a payload at all")); err == nil {
		t.Fatal("expected an error for a body that is neither JSON nor a usable form")
	}
}
