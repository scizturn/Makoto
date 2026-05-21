package voucher

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kyou-id/makoto/internal/domain"
)

func TestKyouClientSendsBirthdayVoucherJSON(t *testing.T) {
	var gotPath string
	var gotToken string
	var gotBody map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotToken = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(`{"voucher_code":"HBD-RUBY-7K2M"}`))
	}))
	defer server.Close()

	client := KyouClient{BaseURL: server.URL, Token: "secret-token", HTTPClient: server.Client()}
	code, err := client.IssueBirthdayVoucher(context.Background(), domain.User{
		ID:    "123",
		Email: "ruby@example.test",
	}, time.Date(2026, 5, 21, 7, 0, 0, 0, time.FixedZone("WIB", 7*60*60)))

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if code != "HBD-RUBY-7K2M" {
		t.Fatalf("expected voucher code, got %q", code)
	}
	if gotPath != "/api/internal/vouchers/birthday" {
		t.Fatalf("expected endpoint path, got %q", gotPath)
	}
	if gotToken != "Bearer secret-token" {
		t.Fatalf("expected bearer token, got %q", gotToken)
	}
	if gotBody["user_id"] != "123" || gotBody["email"] != "ruby@example.test" || gotBody["campaign"] != "birthday-sales" {
		t.Fatalf("unexpected request body: %#v", gotBody)
	}
	if gotBody["expires_at"] != "2026-06-04T00:00:00+07:00" {
		t.Fatalf("expected 2 week expiry, got %q", gotBody["expires_at"])
	}
}
