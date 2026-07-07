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
