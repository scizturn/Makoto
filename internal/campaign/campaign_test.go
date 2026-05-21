package campaign

import (
	"testing"
	"time"

	"github.com/kyou-id/makoto/internal/domain"
)

func TestSelectTemplateUsesRandomChoice(t *testing.T) {
	campaign := BirthdayCampaign{
		TemplateIDs: []string{"tpl_001", "tpl_002", "tpl_003"},
		RandomIntn: func(n int) int {
			if n != 3 {
				t.Fatalf("expected random range 3, got %d", n)
			}
			return 2
		},
	}

	got := campaign.SelectTemplate(time.Date(2026, 5, 21, 7, 0, 0, 0, time.UTC))

	if got != "tpl_003" {
		t.Fatalf("expected tpl_003, got %q", got)
	}
}

func TestBuildMergeDataFallsBackToPopularFYP(t *testing.T) {
	campaign := BirthdayCampaign{
		Closing: "Selamat merayakan hari spesialmu di Kyou!",
	}
	user := domain.User{ID: "user-1", Name: "Ruby", Email: "ruby@example.test"}
	wishlist := []domain.WishlistItem{{ID: "wish-1", Name: "Figure Ruby"}}
	popular := []domain.FYPItem{{ID: "popular-1", Name: "Popular Chara", Kind: "character"}}

	got := campaign.BuildMergeData(user, "HBD-RUBY-7K2M", wishlist, nil, popular)

	if got["name"] != "Ruby" {
		t.Fatalf("expected name Ruby, got %#v", got["name"])
	}
	if got["voucher_code"] != "HBD-RUBY-7K2M" {
		t.Fatalf("expected voucher code, got %#v", got["voucher_code"])
	}
	fypItems, ok := got["fyp_items"].([]domain.FYPItem)
	if !ok {
		t.Fatalf("expected fyp_items []domain.FYPItem, got %T", got["fyp_items"])
	}
	if len(fypItems) != 1 || fypItems[0].ID != "popular-1" {
		t.Fatalf("expected popular fallback, got %#v", fypItems)
	}
}
