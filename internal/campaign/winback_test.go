package campaign

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/kyou-id/makoto/internal/domain"
)

func TestRenderWinbackPastListStripsBracketPrefixesAndLinksItems(t *testing.T) {
	items := []domain.HistoricalItem{
		{
			Name:      "[Exclusive Sale] [with Bonus] PVC Figure Raiden Shogun",
			ImageURL:  "https://kyoucdn.id/items/raiden.jpg.webp",
			URL:       "https://kyou.id/items/192504/",
			OrderDate: time.Date(2026, 3, 12, 0, 0, 0, 0, time.UTC),
		},
	}

	html := RenderWinbackPastListHTML(items)

	if strings.Contains(html, "[Exclusive Sale]") || strings.Contains(html, "[with Bonus]") {
		t.Fatalf("bracket prefixes should be stripped from the name, got: %s", html)
	}
	if !strings.Contains(html, "PVC Figure Raiden Shogun") {
		t.Fatalf("cleaned name missing, got: %s", html)
	}
	if !strings.Contains(html, `href="https://kyou.id/items/192504/"`) {
		t.Fatalf("row should link to the item URL, got: %s", html)
	}
}

func TestRenderWinbackPastListOmitsLinkWhenURLMissing(t *testing.T) {
	items := []domain.HistoricalItem{
		{Name: "Pebble Beanie", ImageURL: "https://kyoucdn.id/items/beanie.jpg.webp"},
	}

	html := RenderWinbackPastListHTML(items)

	if strings.Contains(html, "<a ") {
		t.Fatalf("no link expected when URL is empty, got: %s", html)
	}
	if !strings.Contains(html, "Pebble Beanie") {
		t.Fatalf("item name missing, got: %s", html)
	}
}

func TestRenderWinbackReadyStampsCapsAtSixAndLinks(t *testing.T) {
	items := make([]domain.WishlistItem, 8)
	for i := range items {
		items[i] = domain.WishlistItem{
			ID:       strconv.Itoa(i),
			Name:     "Item " + strconv.Itoa(i),
			URL:      "https://kyou.id/items/" + strconv.Itoa(i) + "/",
			ImageURL: "https://kyoucdn.id/x.webp",
			Price:    320000,
		}
	}
	items[0].IsWishlisted = true

	html := RenderWinbackReadyStampsHTML(items)

	if got := strings.Count(html, "radial-gradient(circle,"+winbackStampMat); got != winbackReadyLimit {
		t.Fatalf("expected %d stamps (2x3), got %d", winbackReadyLimit, got)
	}
	if !strings.Contains(html, "WISHLIST-KU") {
		t.Fatalf("the user's own wishlist item should get the WISHLIST-KU banner")
	}
	if !strings.Contains(html, "KYOU.ID") {
		t.Fatalf("fill items should get the KYOU.ID banner")
	}
	if !strings.Contains(html, ">320K<") {
		t.Fatalf("price should render as a compact denomination, got: %s", html)
	}
	if !strings.Contains(html, `href="https://kyou.id/items/0/"`) {
		t.Fatalf("each stamp should link to its item")
	}
}

func TestRenderWinbackReadyStampsSkipsWakeari(t *testing.T) {
	items := []domain.WishlistItem{
		{ID: "1", Name: "[Wakeari] PVC Figure Lappland", URL: "https://kyou.id/items/1/", ImageURL: "https://x/1.webp", Price: 6950000},
		{ID: "2", Name: "PSA 10 Ludicolo AR", URL: "https://kyou.id/items/2/", ImageURL: "https://x/2.webp", Price: 1150000},
	}

	html := RenderWinbackReadyStampsHTML(items)

	if strings.Contains(html, "https://kyou.id/items/1/") {
		t.Fatalf("wakeari item should be skipped, got: %s", html)
	}
	if !strings.Contains(html, "https://kyou.id/items/2/") {
		t.Fatalf("non-wakeari item should render, got: %s", html)
	}
	if got := strings.Count(html, "radial-gradient(circle,"+winbackStampMat); got != 1 {
		t.Fatalf("expected 1 stamp after skipping wakeari, got %d", got)
	}
}

func TestRenderWinbackReadyStampsEmpty(t *testing.T) {
	if html := RenderWinbackReadyStampsHTML(nil); !strings.Contains(html, "Wishlist kamu lagi kosong") {
		t.Fatalf("empty wishlist should render the fallback message, got: %s", html)
	}
}

func TestStripBracketPrefixesKeepsMidNameBrackets(t *testing.T) {
	got := stripBracketPrefixes("[PSA 10] Ludicolo AR (M2 081/080) [reprint]")
	want := "Ludicolo AR (M2 081/080) [reprint]"
	if got != want {
		t.Fatalf("stripBracketPrefixes = %q, want %q", got, want)
	}
}
