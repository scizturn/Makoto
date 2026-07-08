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
		"WBI8-123",
		8,
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
	// Each item card carries a 2×3 grip marker (6 dots); the first (newest) card is
	// highlighted with an orange border (the rest use the neutral border).
	if got := strings.Count(backInHTML, "border-radius:50%;background:#c9bcb0"); got != 12 {
		t.Fatalf("expected 12 grip dots (6 per item × 2 items), got %d: %s", got, backInHTML)
	}
	if got := strings.Count(backInHTML, "border:2px solid #fc4c02"); got != 1 {
		t.Fatalf("expected exactly 1 orange-bordered (first) item card, got %d: %s", got, backInHTML)
	}
	recoHTML := data["reco_html"].(string)
	if !strings.Contains(recoHTML, "Reco &lt;A&gt;") || !strings.Contains(recoHTML, "Reco B") {
		t.Fatalf("expected reco grid with escaped names: %s", recoHTML)
	}

	// Section must hide when there are no recommendations, even with an anchor.
	hidden := campaign.BuildMergeData(domain.User{Name: "Ruby"}, "WBI8-123", 8, nil,
		domain.WishlistBackInItem{ID: "2", Name: "Purchased Pair"}, nil, "Omatase!")
	if hidden["has_companion"] != false {
		t.Fatalf("expected section hidden without recos, got %v", hidden["has_companion"])
	}
	if data["action_url"] != "https://kyou.id/user/my-voucher?claim=WBI8-123" {
		t.Fatalf("unexpected action URL: %v", data["action_url"])
	}
	if data["voucher_discount"] != 8 || data["has_voucher"] != true {
		t.Fatalf("expected the 8%% tier to surface, got discount=%v has_voucher=%v", data["voucher_discount"], data["has_voucher"])
	}
}

// The coupon block prints the tier, so it must not render when the tier is
// unknown — a job serialized before the tier field existed, or a user whose items
// all sat below the 25% GP floor.
func TestWishlistBackInHidesVoucherWithoutTier(t *testing.T) {
	campaign := WishlistBackInCampaign{ActionURL: "https://kyou.id/user/my-voucher"}
	for _, tc := range []struct {
		name    string
		code    string
		percent int
	}{
		{"no voucher at all", "", 0},
		{"legacy job: code but no tier", "WBIZEZ3HVQLT3F6A", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := campaign.BuildMergeData(domain.User{Name: "Ruby"}, tc.code, tc.percent, nil, domain.WishlistBackInItem{}, nil, "Omatase!")
			if data["has_voucher"] != false {
				t.Fatalf("expected the coupon block suppressed, got has_voucher=%v", data["has_voucher"])
			}
		})
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

func TestCleanItemNameStripsLeadingTags(t *testing.T) {
	cases := map[string]string{
		"[Set of 6] [With Bonus] Honkai: Star Rail Keycap": "Honkai: Star Rail Keycap",
		"[Limited Edition] HSR x MOONDROP Earphone":        "HSR x MOONDROP Earphone",
		"[REVIVE] Figure-rise Standard":                    "Figure-rise Standard",
		"[] Nendoroid Kafka":                               "Nendoroid Kafka",
		"Luminasta Hatsune Miku (18cm)":                    "Luminasta Hatsune Miku (18cm)", // no leading tag, parens kept
		"[Set of 6]":                                       "[Set of 6]",                     // all-tag -> keep original
	}
	for in, want := range cases {
		if got := cleanItemName(in); got != want {
			t.Fatalf("cleanItemName(%q) = %q, want %q", in, got, want)
		}
	}
	// The [revive] badge still keys off the RAW name, not the cleaned one.
	if got := renderStatusBadge(domain.WishlistBackInItem{Name: "[Revive] Figure", Status: "PO"}); !strings.Contains(got, "revive.png") {
		t.Fatalf("revive detection must use raw name, got %s", got)
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
