package email

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/kyou-id/makoto/internal/domain"
)

func TestKirimClientSendsRenderedHTMLWithTransactionalV4(t *testing.T) {
	var gotPath string
	var gotDomain string
	var gotAuthUser string
	var gotAuthPass string
	var gotForm url.Values

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotDomain = r.Header.Get("domain")
		gotAuthUser, gotAuthPass, _ = r.BasicAuth()
		if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/x-www-form-urlencoded") {
			t.Fatalf("expected form content type, got %q", ct)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		gotForm = r.PostForm
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"message_id":"msg_123"}`))
	}))
	defer server.Close()

	client := KirimClient{
		BaseURL:  server.URL,
		Username: "key_test",
		APIToken: "secret_test",
	}

	result, err := client.SendTemplate(context.Background(), domain.EmailMessage{
		Domain:     "kyou.id",
		FromEmail:  "nandayo@kyou.id",
		FromName:   "Kyou.id",
		ToEmail:    "ruby@example.test",
		Subject:    "Selamat ulang tahun, Ruby",
		HTMLBody:   "<h1>Happy birthday Ruby</h1>",
		TextBody:   "Happy birthday Ruby",
		TemplateID: "birthday1.html",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.MessageID != "msg_123" {
		t.Fatalf("expected message id, got %q", result.MessageID)
	}
	if gotPath != "/api/v4/transactional/message" {
		t.Fatalf("unexpected path %q", gotPath)
	}
	if gotDomain != "kyou.id" {
		t.Fatalf("expected domain header, got %q", gotDomain)
	}
	if gotAuthUser != "key_test" || gotAuthPass != "secret_test" {
		t.Fatalf("unexpected basic auth: %q %q", gotAuthUser, gotAuthPass)
	}
	if gotForm.Get("from") != "nandayo@kyou.id" {
		t.Fatalf("unexpected from field: %q", gotForm.Get("from"))
	}
	if gotForm.Get("to") != "ruby@example.test" || gotForm.Get("subject") != "Selamat ulang tahun, Ruby" {
		t.Fatalf("unexpected recipient/subject form: %#v", gotForm)
	}
	if gotForm.Get("html") != "<h1>Happy birthday Ruby</h1>" || gotForm.Get("text") != "Happy birthday Ruby" {
		t.Fatalf("unexpected body form: %#v", gotForm)
	}
}

// Kirim.email's send response carries no message id, so the ONLY thing tying a
// later delivery webhook back to a job is the tag stamped here. If this stops going
// out, every webhook arrives orphaned and silently unmatchable.
func TestKirimClientStampsTheJobIDIntoTags(t *testing.T) {
	var gotForm url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		gotForm = r.PostForm
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"success":true,"message":"Email send jobs dispatched for 1 recipients in 1 job(s)"}`))
	}))
	defer server.Close()

	client := KirimClient{BaseURL: server.URL, Username: "key", APIToken: "secret"}
	_, err := client.SendTemplate(context.Background(), domain.EmailMessage{
		Domain:    "kyou.id",
		FromEmail: "nandayo@kyou.id",
		ToEmail:   "ruby@example.test",
		Subject:   "Selamat ulang tahun, Ruby",
		HTMLBody:  "<h1>Happy birthday</h1>",
		Tags:      "birthday-2026-07-13-user-42",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if got, want := gotForm.Get("tags"), "birthday-2026-07-13-user-42"; got != want {
		t.Fatalf("expected tags=%s, got %q (form: %#v)", want, got, gotForm)
	}
	// `headers` must never be sent. Its validator rejects a string as "must be an
	// array" AND an array as "must be a string" — nothing passes, and sending it
	// 422'd every outgoing email in production.
	for field := range gotForm {
		if strings.HasPrefix(field, "headers") {
			t.Fatalf("the headers field is unusable and 422s every send, got %q", field)
		}
	}
}

// A comma would split the job id into two tags on Kirim.email's side, and the
// webhook would hand back a truncated id that matches nothing.
func TestJobTagStripsCommas(t *testing.T) {
	if got := jobTag("job,with,commas"); got != "jobwithcommas" {
		t.Fatalf("got %s", got)
	}
	if got := jobTag("  "); got != "" {
		t.Fatalf("expected an empty tag, got %q", got)
	}
}

// An untagged message must send no headers field at all — an empty one is still a
// field, and this endpoint validates fields it is given.
func TestKirimClientOmitsTheTagFieldWhenThereIsNoTag(t *testing.T) {
	var gotForm url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		gotForm = r.PostForm
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	client := KirimClient{BaseURL: server.URL, Username: "key", APIToken: "secret"}
	if _, err := client.SendTemplate(context.Background(), domain.EmailMessage{
		Domain: "kyou.id", FromEmail: "nandayo@kyou.id", ToEmail: "ruby@example.test",
		Subject: "x", HTMLBody: "<p>x</p>",
	}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if got := gotForm.Get("tags"); got != "" {
		t.Fatalf("expected no tags field on an untagged message, got %q", got)
	}
}

func TestKirimClientValidatesEmailFromStrictValidationData(t *testing.T) {
	var gotPath string
	var gotAuthUser string
	var gotAuthPass string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuthUser, gotAuthPass, _ = r.BasicAuth()
		if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
			t.Fatalf("expected json content type, got %q", ct)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success":true,"data":{"email":"ruby@example.com","is_valid":true}}`))
	}))
	defer server.Close()

	client := KirimClient{
		BaseURL:  server.URL,
		Username: "validation-user",
		APIToken: "validation-token",
	}

	valid, err := client.Validate(context.Background(), "ruby@example.com")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !valid {
		t.Fatal("expected email to be valid")
	}
	if gotPath != "/api/email/validate/strict" {
		t.Fatalf("unexpected path %q", gotPath)
	}
	if gotAuthUser != "validation-user" || gotAuthPass != "validation-token" {
		t.Fatalf("unexpected basic auth: %q %q", gotAuthUser, gotAuthPass)
	}
}

func TestFailOpenValidatorAllowsProviderErrors(t *testing.T) {
	validator := FailOpenValidator{
		Validator: errorValidator{err: errors.New("provider 500")},
	}

	valid, err := validator.Validate(context.Background(), "ruby@example.com")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !valid {
		t.Fatal("expected fail-open validation to allow email")
	}
}

type errorValidator struct {
	err error
}

func (v errorValidator) Validate(context.Context, string) (bool, error) {
	return false, v.err
}
