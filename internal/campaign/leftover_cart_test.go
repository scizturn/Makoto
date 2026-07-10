package campaign

import (
	"fmt"
	"testing"
	"time"

	"github.com/kyou-id/makoto/internal/domain"
)

func TestLeftoverCartSubjectAlwaysPairsWithItsTemplate(t *testing.T) {
	campaign := LeftoverCartCampaign{
		TemplateIDs: []string{
			"leftover_cart1.html", "leftover_cart2.html", "leftover_cart3.html",
			"leftover_cart4.html", "leftover_cart5.html",
		},
		Subjects: []string{"subject-1", "subject-2", "subject-3", "subject-4", "subject-5"},
	}
	now := time.Date(2026, 5, 29, 0, 0, 0, 0, time.FixedZone("Asia/Jakarta", 7*60*60))

	pairs := map[string]string{
		"leftover_cart1.html": "subject-1",
		"leftover_cart2.html": "subject-2",
		"leftover_cart3.html": "subject-3",
		"leftover_cart4.html": "subject-4",
		"leftover_cart5.html": "subject-5",
	}
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("leftover-cart-2026-05-29-user-%d", i)
		tmpl := campaign.SelectTemplate(now, key)
		subject := campaign.SelectSubject(now, key)
		if want := pairs[tmpl]; subject != want {
			t.Fatalf("template %s drew subject %q, want %q", tmpl, subject, want)
		}
	}
}

func TestLeftoverCartGreetingDoesNotFollowTheTemplate(t *testing.T) {
	slots := []string{"1", "2", "3", "4", "5"}
	campaign := LeftoverCartCampaign{TemplateIDs: slots, Greetings: slots}
	now := time.Date(2026, 5, 29, 0, 0, 0, 0, time.FixedZone("Asia/Jakarta", 7*60*60))

	// The greeting seed carries a "_greeting" suffix, so unlike the subject it
	// must not track the template index.
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("leftover-cart-user-%d", i)
		if campaign.SelectTemplate(now, key) != campaign.SelectGreeting(now, key) {
			return
		}
	}
	t.Fatal("greeting matched the template slot for all 100 keys; the _greeting seed offset is gone")
}

func TestLeftoverCartRenderSubjectFillsFirstName(t *testing.T) {
	campaign := LeftoverCartCampaign{}
	got := campaign.RenderSubject("Psst {{ .FirstName }}, keranjangmu nunggu!", domain.User{Name: "Bimo Tyastomo"})
	if want := "Psst Bimo, keranjangmu nunggu!"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
