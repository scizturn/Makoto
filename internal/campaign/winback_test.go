package campaign

import (
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

func TestStripBracketPrefixesKeepsMidNameBrackets(t *testing.T) {
	got := stripBracketPrefixes("[PSA 10] Ludicolo AR (M2 081/080) [reprint]")
	want := "Ludicolo AR (M2 081/080) [reprint]"
	if got != want {
		t.Fatalf("stripBracketPrefixes = %q, want %q", got, want)
	}
}
