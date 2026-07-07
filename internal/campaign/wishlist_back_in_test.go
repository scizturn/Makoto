package campaign

import (
	"strings"
	"testing"

	"github.com/kyou-id/makoto/internal/domain"
)

func TestWishlistBackInMergeDataEscapesItemsAndHandlesCompanion(t *testing.T) {
	campaign := WishlistBackInCampaign{ActionURL: "https://kyou.id/user/my-voucher", Closing: "Sampai ketemu!"}
	recos := []domain.WishlistBackInItem{
		{ID: "10", Name: "Reco <A>", URL: "https://kyou.id/items/10/", Price: 500000},
		{ID: "11", Name: "Reco B", URL: "https://kyou.id/items/11/", Price: 600000},
	}
	data := campaign.BuildMergeData(
		domain.User{Name: "Ruby Reinze"},
		"WBI123",
		[]domain.WishlistBackInItem{
			{ID: "1", Name: "Figure <Ready>", URL: "https://kyou.id/items/1/", Price: 100000},
			{ID: "3", Name: "Second Item", URL: "https://kyou.id/items/3/", Price: 250000},
		},
		domain.WishlistBackInItem{ID: "2", Name: "Purchased Pair", URL: "https://kyou.id/items/2/"},
		recos,
		"Omatase, Ruby!",
	)
	if data["first_name"] != "Ruby" || data["has_companion"] != true || data["item_count"] != 2 {
		t.Fatalf("unexpected merge data: %#v", data)
	}
	backInHTML := data["back_in_item_html"].(string)
	if strings.Contains(backInHTML, "Figure <Ready>") || !strings.Contains(backInHTML, "Figure &lt;Ready&gt;") {
		t.Fatalf("expected escaped item HTML: %s", backInHTML)
	}
	if !strings.Contains(backInHTML, "Second Item") {
		t.Fatalf("expected all items rendered as rows: %s", backInHTML)
	}
	if !strings.Contains(backInHTML, ">01<") || !strings.Contains(backInHTML, ">02<") {
		t.Fatalf("expected numbered index rows 01 and 02: %s", backInHTML)
	}
	recoHTML := data["reco_html"].(string)
	if !strings.Contains(recoHTML, "Reco &lt;A&gt;") || !strings.Contains(recoHTML, "Reco B") {
		t.Fatalf("expected reco grid with escaped names: %s", recoHTML)
	}

	// Section must hide when there are no recommendations, even with an anchor.
	hidden := campaign.BuildMergeData(domain.User{Name: "Ruby"}, "WBI123", nil,
		domain.WishlistBackInItem{ID: "2", Name: "Purchased Pair"}, nil, "Omatase!")
	if hidden["has_companion"] != false {
		t.Fatalf("expected section hidden without recos, got %v", hidden["has_companion"])
	}
	if data["action_url"] != "https://kyou.id/user/my-voucher?claim=WBI123" {
		t.Fatalf("unexpected action URL: %v", data["action_url"])
	}
}

func TestWishlistBackInStatusBadgeMatchesHanamaru(t *testing.T) {
	cases := []struct{ status, name, wantColor, wantLabel string }{
		{"ready", "Figure X", "#41b774", "Ready Stock"},
		{"PO", "Figure Y", "#657996", "Pre-Order"},
		{"LPO", "Figure Z", "#d3647a", "Late Pre-Order"},
		{"BO", "Figure W", "#996291", "Back Order"},
	}
	for _, c := range cases {
		got := renderStatusBadge(domain.WishlistBackInItem{Status: c.status, Name: c.name})
		if !strings.Contains(got, "background:"+c.wantColor) || !strings.Contains(got, c.wantLabel) {
			t.Fatalf("status %q: want bg %s + label %q, got %s", c.status, c.wantColor, c.wantLabel, got)
		}
	}
	// Revive is a name tag; it wins over status and renders the shared image.
	if got := renderStatusBadge(domain.WishlistBackInItem{Status: "PO", Name: "[Revive] Figure"}); !strings.Contains(got, "revive.png") {
		t.Fatalf("expected revive image badge, got %s", got)
	}
}

func TestWishlistBackInPrice(t *testing.T) {
	// Discount: discounted price (brand color) + original as struck sub.
	main, color, sub, struck := wishlistBackInPrice(domain.WishlistBackInItem{Price: 665000, DiscountPrice: 585000})
	if main != "IDR 585.000" || color != "#fc4c02" || sub != "665.000" || !struck {
		t.Fatalf("discount: main=%q color=%q sub=%q struck=%v", main, color, sub, struck)
	}
	// DP (PO): "DP IDR <dp>" + "/ <full>".
	main, _, sub, struck = wishlistBackInPrice(domain.WishlistBackInItem{Price: 2100000, DownPayment: 450000, Status: "PO"})
	if main != "DP IDR 450.000" || sub != "/ 2.100.000" || struck {
		t.Fatalf("dp: main=%q sub=%q struck=%v", main, sub, struck)
	}
	// Plain.
	main, color, sub, _ = wishlistBackInPrice(domain.WishlistBackInItem{Price: 300000})
	if main != "IDR 300.000" || color != "#2a2a2a" || sub != "" {
		t.Fatalf("plain: main=%q color=%q sub=%q", main, color, sub)
	}
}
