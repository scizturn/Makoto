package campaign

import (
	"strings"
	"testing"

	"github.com/kyou-id/makoto/internal/domain"
)

func TestWishlistBackInMergeDataEscapesItemsAndHandlesCompanion(t *testing.T) {
	campaign := WishlistBackInCampaign{ActionURL: "https://kyou.id/user/my-voucher", Closing: "Sampai ketemu!"}
	data := campaign.BuildMergeData(
		domain.User{Name: "Ruby Reinze"},
		"WBI123",
		domain.WishlistBackInItem{ID: "1", Name: "Figure <Ready>", URL: "https://kyou.id/items/1/", Price: 100000},
		domain.WishlistBackInItem{ID: "2", Name: "Purchased Pair", URL: "https://kyou.id/items/2/"},
		"Omatase, Ruby!",
	)
	if data["first_name"] != "Ruby" || data["has_companion"] != true {
		t.Fatalf("unexpected merge data: %#v", data)
	}
	if strings.Contains(data["back_in_item_html"].(string), "Figure <Ready>") || !strings.Contains(data["back_in_item_html"].(string), "Figure &lt;Ready&gt;") {
		t.Fatalf("expected escaped item HTML: %s", data["back_in_item_html"])
	}
	if data["action_url"] != "https://kyou.id/user/my-voucher?claim=WBI123" {
		t.Fatalf("unexpected action URL: %v", data["action_url"])
	}
}
