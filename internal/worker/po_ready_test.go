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
		Subjects:    []string{"{{ .FirstName }}, PO kamu ready!"},
		Greetings:   []string{"Omatase, {{ .FirstName }}!"},
		HistoryURL:  "https://kyou.id/user/history",
		Closing:     "closing",
		RandomIntn:  func(int) int { return 0 },
	}
	p := NewPoReadyProcessor(sender, fakeValidator{valid: true}, c)
	p.Renderer = fakeRenderer{subject: "fallback", html: "<html></html>"}
	return p
}

func poReadyJob() domain.PoReadyJob {
	return domain.PoReadyJob{
		ID:        "po-ready-2026-07-01-order-555",
		OrderID:   "555",
		UserID:    "1",
		Date:      time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		User:      domain.User{ID: "1", Name: "Budi Santoso", Email: "budi@example.test", IsActive: true},
		Items:     []domain.PoReadyItem{{ID: "10", Name: "Figure", Price: 350000, Quantity: 1}},
		Remaining: 250000,
		Attempt:   1,
	}
}

func TestPoReadyProcessorSendsAndPassesOrderMergeData(t *testing.T) {
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
	if data["order_id"] != "555" || data["remaining_text"] != "IDR 250.000" {
		t.Fatalf("unexpected merge data: order_id=%v remaining=%v", data["order_id"], data["remaining_text"])
	}
	if !strings.Contains(result.Subject, "Budi") {
		t.Fatalf("subject should use first name, got %q", result.Subject)
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
