package worker

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/kyou-id/makoto/internal/campaign"
	"github.com/kyou-id/makoto/internal/domain"
)

func newPoReadyProcessor(sender *fakeSender) *PoReadyProcessor {
	c := campaign.PoReadyCampaign{
		TemplateIDs: []string{"po_ready1.html"},
		Subjects:    []string{"{{ .FirstName }}, wishlist kamu ready!"},
		Greetings:   []string{"Omatase, {{ .FirstName }}!"},
		WishlistURL: "https://kyou.id/user/wishlist",
		Closing:     "closing",
		RandomIntn:  func(int) int { return 0 },
	}
	p := NewPoReadyProcessor(sender, fakeValidator{valid: true}, c)
	p.Renderer = fakeRenderer{subject: "fallback", html: "<html></html>"}
	return p
}

func poReadyJob() domain.PoReadyJob {
	return domain.PoReadyJob{
		ID:     "po-ready-2026-07-10-user-1",
		UserID: "1",
		Date:   time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC),
		User:   domain.User{ID: "1", Name: "Budi Santoso", Email: "budi@example.test", IsActive: true},
		Items: []domain.PoReadyItem{
			{ID: "10", Name: "Figure", URL: "https://kyou.id/items/10/", Price: 350000},
		},
		Attempt: 1,
	}
}

func TestPoReadyProcessorSendsWithWishlistMergeData(t *testing.T) {
	sender := &fakeSender{}
	p := newPoReadyProcessor(sender)

	result, err := p.Process(context.Background(), poReadyJob())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Outcome != ProcessOutcomeSent {
		t.Fatalf("expected sent, got %q (%s)", result.Outcome, result.SkipReason)
	}
	if len(sender.messages) != 1 {
		t.Fatalf("expected one send, got %d", len(sender.messages))
	}
	data := sender.messages[0].SubstitutionData
	if data["action_url"] != "https://kyou.id/user/wishlist" {
		t.Fatalf("CTA must point at the wishlist, got %v", data["action_url"])
	}
	if data["item_count"] != "1" {
		t.Fatalf("unexpected item_count: %v", data["item_count"])
	}
	// The pelunasan campaign's billing fields must be gone from the contract.
	for _, gone := range []string{"order_id", "remaining_text", "down_payment_text", "eta"} {
		if _, ok := data[gone]; ok {
			t.Fatalf("merge data still carries billing field %q", gone)
		}
	}
	if !strings.Contains(result.Subject, "Budi") {
		t.Fatalf("subject should use first name, got %q", result.Subject)
	}
}

func TestPoReadyProcessorCountsOnlyRenderedItems(t *testing.T) {
	sender := &fakeSender{}
	p := newPoReadyProcessor(sender)

	job := poReadyJob()
	job.Items = nil
	for i := 0; i < 8; i++ {
		job.Items = append(job.Items, domain.PoReadyItem{ID: "x", Name: "Figure", Price: 1000})
	}
	if _, err := p.Process(context.Background(), job); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// item_count promises what the grid actually shows, not what the job carried.
	if got := sender.messages[0].SubstitutionData["item_count"]; got != "5" {
		t.Fatalf("expected item_count clamped to the rendered cards, got %v", got)
	}
}

func TestPoReadyProcessorRendersDiscountStrikethrough(t *testing.T) {
	sender := &fakeSender{}
	p := newPoReadyProcessor(sender)

	job := poReadyJob()
	job.Items[0].Price = 350000
	job.Items[0].DiscountPrice = 280000
	if _, err := p.Process(context.Background(), job); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	html, _ := sender.messages[0].SubstitutionData["items_html"].(string)
	if !strings.Contains(html, "line-through") || !strings.Contains(html, "IDR 280.000") {
		t.Fatalf("expected discounted price with struck-through original, got %q", html)
	}
}

func TestPoReadyProcessorSkipsInactiveAndEmpty(t *testing.T) {
	p := newPoReadyProcessor(&fakeSender{})

	inactive := poReadyJob()
	inactive.User.IsActive = false
	if r, _ := p.Process(context.Background(), inactive); r.Outcome != ProcessOutcomeSkipped || r.SkipReason != "inactive_user" {
		t.Fatalf("expected inactive_user skip, got %#v", r)
	}

	noItems := poReadyJob()
	noItems.Items = nil
	if r, _ := p.Process(context.Background(), noItems); r.Outcome != ProcessOutcomeSkipped || r.SkipReason != "no_ready_items" {
		t.Fatalf("expected no_ready_items skip, got %#v", r)
	}
}
