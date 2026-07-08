package worker

import (
	"context"
	"testing"
	"time"

	"github.com/kyou-id/makoto/internal/campaign"
	"github.com/kyou-id/makoto/internal/domain"
)

// The renderer supplies a fallback subject; the campaign's per-template Subjects
// must win, be rendered with the user's first name, and reach the sent message.
func TestWishlistBackInProcessorUsesPerTemplateSubject(t *testing.T) {
	sender := &fakeSender{}
	c := campaign.WishlistBackInCampaign{
		TemplateIDs: []string{"wishlist_back_in1.html", "wishlist_back_in2.html", "wishlist_back_in3.html"},
		Subjects: []string{
			"Kabar baik untuk {{ .FirstName }}! Wishlist kamu sudah tersedia kembali!",
			"Surprise! Item inceran {{ .FirstName }} tersedia kembali hari ini",
			"Yatta! Penantianmu berakhir, wishlist {{ .FirstName }} sudah bisa di-checkout lagi!",
		},
		ActionURL: "https://kyou.id/user/my-voucher",
	}
	p := NewWishlistBackInProcessor(sender, fakeValidator{valid: true}, c)
	p.Renderer = fakeRenderer{subject: "FALLBACK SUBJECT", html: "<html></html>"}

	job := domain.WishlistBackInJob{
		ID:                     "wishlist-back-in-2026-07-10-user-42",
		Date:                   time.Date(2026, 7, 10, 9, 0, 0, 0, time.UTC),
		User:                   domain.User{Name: "Ruby Reinze", Email: "ruby@example.test", IsActive: true},
		VoucherCode:            "WBI8-ABC",
		VoucherDiscountPercent: 8,
		Items:                  []domain.WishlistBackInItem{{ID: "1", Name: "Figure"}},
	}
	result, err := p.Process(context.Background(), job)
	if err != nil {
		t.Fatal(err)
	}
	if result.Subject == "FALLBACK SUBJECT" {
		t.Fatal("the renderer's fallback subject leaked; per-template subject was not applied")
	}

	// Whichever template the seed picked, the subject must be that template's.
	want := map[string]string{
		"wishlist_back_in1.html": "Kabar baik untuk Ruby! Wishlist kamu sudah tersedia kembali!",
		"wishlist_back_in2.html": "Surprise! Item inceran Ruby tersedia kembali hari ini",
		"wishlist_back_in3.html": "Yatta! Penantianmu berakhir, wishlist Ruby sudah bisa di-checkout lagi!",
	}[result.TemplateID]
	if result.Subject != want {
		t.Fatalf("template %s: subject = %q, want %q", result.TemplateID, result.Subject, want)
	}
	if len(sender.messages) != 1 || sender.messages[0].Subject != want {
		t.Fatalf("sent message subject = %q, want %q", sender.messages[0].Subject, want)
	}
}

// With no Subjects configured the renderer's single subject must survive.
func TestWishlistBackInProcessorFallsBackToRendererSubject(t *testing.T) {
	sender := &fakeSender{}
	c := campaign.WishlistBackInCampaign{TemplateIDs: []string{"wishlist_back_in1.html"}}
	p := NewWishlistBackInProcessor(sender, fakeValidator{valid: true}, c)
	p.Renderer = fakeRenderer{subject: "Ruby, wishlist kamu tersedia lagi!", html: "<html></html>"}

	result, err := p.Process(context.Background(), domain.WishlistBackInJob{
		ID:   "job-1",
		Date: time.Date(2026, 7, 10, 9, 0, 0, 0, time.UTC),
		User: domain.User{Name: "Ruby", Email: "ruby@example.test", IsActive: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Subject != "Ruby, wishlist kamu tersedia lagi!" {
		t.Fatalf("got %q", result.Subject)
	}
}
