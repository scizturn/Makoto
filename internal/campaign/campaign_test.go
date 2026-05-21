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
		Closing:   "Selamat merayakan hari spesialmu di Kyou!",
		ActionURL: "https://kyou.id/account/vouchers",
	}
	user := domain.User{ID: "user-1", Name: "Ruby", Email: "ruby@example.test"}
	wishlist := []domain.WishlistItem{{ID: "wish-1", Name: "Figure <Ruby>", URL: "https://kyou.id/items/1?x=<bad>"}}
	popular := []domain.FYPItem{{ID: "popular-1", Name: "Popular <Chara>", Kind: "character"}}

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
	if got["action_url"] != "https://kyou.id/account/vouchers" {
		t.Fatalf("expected action_url, got %#v", got["action_url"])
	}
	wishlistHTML, ok := got["wishlist_html"].(string)
	if !ok {
		t.Fatalf("expected wishlist_html string, got %T", got["wishlist_html"])
	}
	if wishlistHTML == "" || containsAny(wishlistHTML, []string{"Figure <Ruby>", "?x=<bad>"}) {
		t.Fatalf("expected escaped wishlist html, got %q", wishlistHTML)
	}
	if !containsAll(wishlistHTML, []string{"Figure &lt;Ruby&gt;", "https://kyou.id/items/1?x=&lt;bad&gt;"}) {
		t.Fatalf("expected wishlist html content, got %q", wishlistHTML)
	}
	fypHTML, ok := got["fyp_html"].(string)
	if !ok {
		t.Fatalf("expected fyp_html string, got %T", got["fyp_html"])
	}
	if !containsAll(fypHTML, []string{"Popular &lt;Chara&gt;", "character"}) {
		t.Fatalf("expected fyp html content, got %q", fypHTML)
	}
}

func containsAll(value string, needles []string) bool {
	for _, needle := range needles {
		if !contains(value, needle) {
			return false
		}
	}
	return true
}

func containsAny(value string, needles []string) bool {
	for _, needle := range needles {
		if contains(value, needle) {
			return true
		}
	}
	return false
}

func contains(value string, needle string) bool {
	for i := 0; i+len(needle) <= len(value); i++ {
		if value[i:i+len(needle)] == needle {
			return true
		}
	}
	return needle == ""
}
