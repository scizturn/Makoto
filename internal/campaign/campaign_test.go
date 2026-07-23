package campaign

import (
	"fmt"
	"os"
	"strings"
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

	got := campaign.SelectTemplate(time.Date(2026, 5, 21, 7, 0, 0, 0, time.UTC), "job-1")

	if got != "tpl_003" {
		t.Fatalf("expected tpl_003, got %q", got)
	}
}

func TestBirthdaySubjectAlwaysPairsWithItsTemplate(t *testing.T) {
	campaign := BirthdayCampaign{
		TemplateIDs: []string{"birthday1.html", "birthday2.html", "birthday3.html"},
		Subjects:    []string{"subject-1", "subject-2", "subject-3"},
	}
	now := time.Date(2026, 5, 29, 0, 0, 0, 0, time.FixedZone("Asia/Jakarta", 7*60*60))

	pairs := map[string]string{
		"birthday1.html": "subject-1",
		"birthday2.html": "subject-2",
		"birthday3.html": "subject-3",
	}
	for i := 0; i < 60; i++ {
		key := fmt.Sprintf("birthday-2026-05-29-user-%d", i)
		tmpl := campaign.SelectTemplate(now, key)
		subject := campaign.SelectSubject(now, key)
		if want := pairs[tmpl]; subject != want {
			t.Fatalf("template %s drew subject %q, want %q", tmpl, subject, want)
		}
	}
}

func TestBirthdayRenderSubjectFillsName(t *testing.T) {
	campaign := BirthdayCampaign{}
	got, err := campaign.RenderSubject("Hore, {{ .Name }}! Saatnya Ambil Hadiah Ulang Tahunmu 🎂", domain.User{Name: "Bimo Tyastomo"})
	if err != nil {
		t.Fatalf("render subject: %v", err)
	}
	if want := "Hore, Bimo Tyastomo! Saatnya Ambil Hadiah Ulang Tahunmu 🎂"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestSelectTemplateUsesStablePerJobSeed(t *testing.T) {
	campaign := BirthdayCampaign{TemplateIDs: []string{"tpl_001", "tpl_002", "tpl_003"}}
	now := time.Date(2026, 5, 29, 0, 0, 0, 0, time.FixedZone("Asia/Jakarta", 7*60*60))

	first := campaign.SelectTemplate(now, "birthday-2026-05-29-user-147044")
	second := campaign.SelectTemplate(now, "birthday-2026-05-29-user-147044")
	if first != second {
		t.Fatalf("expected stable template for same job key, got %q then %q", first, second)
	}

	seen := map[string]bool{}
	for _, key := range []string{
		"birthday-2026-05-29-user-147044",
		"birthday-2026-05-29-user-147045",
		"birthday-2026-05-29-user-147046",
		"birthday-2026-05-29-user-147047",
		"birthday-2026-05-29-user-147048",
		"birthday-2026-05-29-user-147049",
	} {
		seen[campaign.SelectTemplate(now, key)] = true
	}
	if len(seen) < 2 {
		t.Fatalf("expected job keys to spread across templates, got %#v", seen)
	}
}

func TestBuildMergeDataFallsBackToPopularFYP(t *testing.T) {
	campaign := BirthdayCampaign{
		Closing:   "Selamat merayakan hari spesialmu di Kyou!",
		ActionURL: "https://kyou.id/user/my-voucher",
	}
	user := domain.User{ID: "user-1", Name: "Ruby", Email: "ruby@example.test"}
	wishlist := []domain.WishlistItem{{ID: "wish-1", Name: "Figure <Ruby>", URL: "https://kyou.id/items/1/"}}
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
	if got["action_url"] != "https://kyou.id/user/my-voucher?claim=HBD-RUBY-7K2M" {
		t.Fatalf("expected action_url, got %#v", got["action_url"])
	}
	wishlistHTML, ok := got["wishlist_html"].(string)
	if !ok {
		t.Fatalf("expected wishlist_html string, got %T", got["wishlist_html"])
	}
	if wishlistHTML == "" || containsAny(wishlistHTML, []string{"Figure <Ruby>"}) {
		t.Fatalf("expected escaped wishlist html, got %q", wishlistHTML)
	}
	if !containsAll(wishlistHTML, []string{"Figure &lt;Ruby&gt;", "https://kyou.id/items/1/"}) {
		t.Fatalf("expected wishlist html content, got %q", wishlistHTML)
	}
	fypHTML, ok := got["fyp_html"].(string)
	if !ok {
		t.Fatalf("expected fyp_html string, got %T", got["fyp_html"])
	}
	if !containsAll(fypHTML, []string{"Wishlist", "&lt;Chara&gt;"}) {
		t.Fatalf("expected fyp html content, got %q", fypHTML)
	}
}

func TestBuildMergeDataAppendsClaimToExistingActionURLQuery(t *testing.T) {
	campaign := BirthdayCampaign{ActionURL: "https://kyou.id/user/my-voucher?utm=email"}
	user := domain.User{ID: "user-1", Name: "Ruby", Email: "ruby@example.test"}

	got := campaign.BuildMergeData(user, "HBD RUBY/7K2M", nil, nil, nil)

	if got["action_url"] != "https://kyou.id/user/my-voucher?claim=HBD+RUBY%2F7K2M&utm=email" {
		t.Fatalf("expected encoded claim action_url, got %#v", got["action_url"])
	}
}

func TestBuildMergeDataTopsUpPartialFYPWithPopularItems(t *testing.T) {
	campaign := BirthdayCampaign{}
	user := domain.User{ID: "user-1", Name: "Ruby", Email: "ruby@example.test"}
	fyp := []domain.FYPItem{{ID: "fyp-1", Name: "Dorothy", Kind: "character"}}
	popular := []domain.FYPItem{
		{ID: "popular-1", Name: "Frieren", Kind: "series"},
		{ID: "popular-2", Name: "Miku", Kind: "series"},
		{ID: "popular-3", Name: "Extra", Kind: "series"},
	}

	got := campaign.BuildMergeData(user, "HBD", nil, fyp, popular)

	fypItems, ok := got["fyp_items"].([]domain.FYPItem)
	if !ok {
		t.Fatalf("expected fyp_items []domain.FYPItem, got %T", got["fyp_items"])
	}
	if len(fypItems) != 3 {
		t.Fatalf("expected partial fyp to be topped up to 3 items, got %#v", fypItems)
	}
	if fypItems[0].ID != "fyp-1" || fypItems[1].ID != "popular-1" || fypItems[2].ID != "popular-2" {
		t.Fatalf("expected fyp then popular top-up order, got %#v", fypItems)
	}
	fypHTML, ok := got["fyp_html"].(string)
	if !ok {
		t.Fatalf("expected fyp_html string, got %T", got["fyp_html"])
	}
	if countOccurrences(fypHTML, `<td width="230"`) != 3 {
		t.Fatalf("expected three rendered cards, got %q", fypHTML)
	}
}

func TestRenderWishlistHTMLUsesImageCardsWhenImageURLExists(t *testing.T) {
	got := RenderWishlistHTML([]domain.WishlistItem{{
		ID:           "wish-1",
		Name:         "Vivian Banshee Mockingbird Q Series Acrylic Keychain - Zenless Zone Zero (7,4cm)",
		URL:          "https://kyou.id/items/1/",
		ImageURL:     "https://kyoucdn.id/items/figure.jpg.webp?x=<bad>",
		Price:        850000,
		Status:       "ready",
		Manufacturer: "Vocaloid",
		SeriesName:   "Zenless Zone Zero",
	}})

	if !containsAll(got, []string{
		`<img src="https://kyoucdn.id/items/figure.jpg.webp?x=&lt;bad&gt;"`,
		`alt="Vivian Banshee Mockingbird Q"`,
		`Ready Stock`,
		`background:#40b774`,
		`Wishlist Pick`,
		`Vivian Banshee Mockingbird Q`,
		`ZENLESS ZONE ZERO`,
		`margin:0 auto;width:600px;padding:26px 30px`,
		`td width="170" valign="middle" style="width:170px;padding:0 42px 0 0;`,
		`td width="404" valign="middle" style="width:404px;padding:0 0 0 0;`,
		`margin:0 0 4px;color:#ffd68a;font-size:14px;font-weight:900;letter-spacing:3px;text-transform:uppercase;`,
		`margin:0 0 16px;color:#f7dfce;font-size:16px;font-weight:650;line-height:1.52;`,
		`font-size:19px;font-weight:800;line-height:1.22`,
		`<a href="https://kyou.id/items/1/" style="display:block;color:inherit;text-decoration:none;">`,
	}) {
		t.Fatalf("expected wishlist image card html, got %q", got)
	}
	if containsAny(got, []string{"Vivian Banshee Mockingbird Q Series Acrylic Keychain", "Zenless Zone Zero (7,4cm)", `font-style:italic`, `font-size:22px;font-weight:900;line-height:1.18`, `style="color:#ffffff;text-decoration:none;"`}) {
		t.Fatalf("expected escaped image card html, got %q", got)
	}
}

func TestAbbreviateCardTitleKeepsCharacterBeforeQSeries(t *testing.T) {
	name, version := abbreviateCardTitle(
		"Vivian Banshee Mockingbird Q Series Acrylic Keychain - Zenless Zone Zero (7,4cm)",
		"Zenless Zone Zero",
	)
	if name != "Vivian Banshee Mockingbird" || version != "" {
		t.Fatalf("unexpected Q Series title: name=%q version=%q", name, version)
	}
}

func TestDiscountedWishlistDisplayNamePrefersCharacterName(t *testing.T) {
	name, version := discountedWishlistDisplayName(domain.DiscountedWishlistItem{
		Name:          "Vivian Banshee Mockingbird Q Series Acrylic Keychain - Zenless Zone Zero",
		CharacterName: "Vivian Banshee Mockingbird",
		SeriesName:    "Zenless Zone Zero",
	})
	if name != "Vivian Banshee Mockingbird" || version != "" {
		t.Fatalf("unexpected discounted wishlist display name: name=%q version=%q", name, version)
	}
}

func TestDiscountedPromoGridRendersTapedWishlistCards(t *testing.T) {
	got := RenderDiscountedPromoGridHTML([]domain.DiscountedWishlistItem{{
		ID: "1", CharacterName: "Mihono Bourbon", URL: "https://kyou.id/items/1/",
		ImageURL: "https://kyoucdn.id/items/1.webp", SeriesName: "Uma Musume", OriginalPrice: 400000, DiscountPrice: 300000, IsWishlisted: true,
	}}, nil)
	if !containsAll(got, []string{"repeating-linear-gradient", "transform:rotate(-1.4deg)", "&hearts; WISHLIST", "2px solid #ff5a24", "Uma Musume", "Mihono Bourbon", "font-size:12px", "IDR 400.000", "text-decoration:line-through", "font-size:13px", "IDR 300.000", "-25%"}) {
		t.Fatalf("expected taped wishlist promo card, got %q", got)
	}
}

func TestDiscountedWishlistFooterOmitsSelfRenderedUnsubscribeLink(t *testing.T) {
	// Unsubscribe is handled by Kirim.email (List-Unsubscribe / provider footer),
	// so Makoto must not render its own unsubscribe link into the body.
	c := DiscountedWishlistCampaign{}
	data := c.BuildMergeData(domain.User{Name: "Budi"}, nil, "")
	footer, _ := data["footer_html"].(string)
	if strings.Contains(footer, "Atur preferensi email") {
		t.Fatalf("expected no self-rendered unsubscribe link, got %q", footer)
	}
}

func TestRenderFYPHTMLUsesImageCardsWhenImageURLExists(t *testing.T) {
	got := RenderFYPHTML([]domain.FYPItem{
		{
			ID:         "fyp-1",
			Name:       "PVC Figure Gift+ 1/8 Sunday - Star Rail LIVE Ver. Honkai: Star Rail",
			Kind:       "character",
			ImageURL:   "https://kyoucdn.id/items/chara.jpg.webp",
			Price:      150000,
			Status:     "PO",
			SeriesName: "Honkai: Star Rail",
		},
		{
			ID:         "fyp-2",
			Name:       "[Random] Hatsune Miku Stellar Voice Series Can Badge Blind Box - Vocaloid",
			Kind:       "series",
			ImageURL:   "https://kyoucdn.id/items/series.jpg.webp",
			Price:      650000,
			Status:     "ready",
			SeriesName: "Vocaloid",
		},
		{
			ID:         "fyp-3",
			Name:       "[Mono Goods] Protect Me Umbrella - Wind Breaker",
			Kind:       "series",
			ImageURL:   "https://kyoucdn.id/items/another.jpg.webp",
			Price:      850000,
			Status:     "ready",
			SeriesName: "Wind Breaker",
		},
	})

	if !containsAll(got, []string{
		`<table role="presentation"`,
		`width="690"`,
		`align="center"`,
		`<td width="230"`,
		`width:180px;height:287px`,
		`background:url('https://kyoucdn.id/static/assets/item%20bg1.jpg') center/cover no-repeat;`,
		`<img src="https://kyoucdn.id/items/chara.jpg.webp"`,
		`alt="1/8 Sunday"`,
		`1/8 Sunday`,
		`Star Rail LIVE Ver`,
		`https://kyou.id/items/fyp-1/`,
		`https://kyoucdn.id/static/assets/PO_CTA_Email.png`,
		`alt="Cek item"`,
		`Honkai: Star Rail`,
		`Vocaloid`,
		`Miku`,
		`Wind Breaker`,
		`Umbrella`,
	}) {
		t.Fatalf("expected fyp image card html, got %q", got)
	}
	if containsAny(got, []string{"IDR 150.000", "IDR 650.000", "IDR 850.000", `font-style:italic`, `PVC Figure`, `Gift+`, `[Random]`, `Can Badge Blind Box`, `[Mono Goods]`, `Ready Stock`, `Pre-Order`, `Late Pre-Order`, `background:#40b774`, `background:#657996`, `background:#d3647a`}) {
		t.Fatalf("expected fyp cards to omit prices and product noise, got %q", got)
	}
	if countOccurrences(got, `<a href="https://kyou.id/items/`) != 3 {
		t.Fatalf("expected every fyp card to be wrapped in a link, got %q", got)
	}
	if containsAny(got, []string{"display:grid", "display:flex"}) {
		t.Fatalf("expected table layout without grid/flex, got %q", got)
	}
	if countOccurrences(got, `<td width="230"`) != 3 {
		t.Fatalf("expected exactly three table cells, got %q", got)
	}
	if countOccurrences(got, `https://kyoucdn.id/static/assets/PO_CTA_Email.png`) != 3 {
		t.Fatalf("expected one image button per card, got %q", got)
	}
}

func TestFYPHTMLDoesNotRenderStatusBadges(t *testing.T) {
	got := RenderFYPHTML([]domain.FYPItem{
		{ID: "po", Name: "PO Pick", Status: "PO", PODeadline: ptrTime(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC))},
		{ID: "ready", Name: "Ready Pick", Status: "ready"},
		{ID: "lpo", Name: "LPO Pick", Status: "PO"},
	})

	if containsAny(got, []string{`Pre-Order`, `Ready Stock`, `Late Pre-Order`, `>PO<`, `>READY<`, `>LPO<`, `background:#657996`, `background:#40b774`, `background:#d3647a`}) {
		t.Fatalf("expected fyp cards without status badges, got %q", got)
	}
}

func TestRenderCartItemsHTMLIncludesBrowserCartButton(t *testing.T) {
	got := RenderCartItemsHTMLWithURL([]domain.WishlistItem{
		{
			ID:         "item-1",
			Name:       "Nendoroid Rin <Touring>",
			URL:        "https://kyou.id/items/1/?x=<bad>",
			ImageURL:   "https://kyoucdn.id/items/rin.jpg.webp?x=<bad>",
			Price:      735000,
			Status:     "PO",
			SeriesName: "Laid-Back Camp",
		},
		{
			ID:         "item-1",
			Name:       "Nendoroid Rin <Touring>",
			Price:      735000,
			Status:     "ready",
			SeriesName: "Laid-Back Camp",
		},
		{
			ID:         "item-2",
			Name:       "Nami Can Badge",
			Price:      185000,
			Status:     "FLASH PO",
			SeriesName: "One Piece",
		},
	}, "https://kyou.id/user/cart?utm=<email>")

	if !containsAll(got, []string{
		`https://kyou.id/user/cart`,
		`Shopping Cart`,
		`3 item &bull; 2 produk`,
		`Lanjut ke Keranjang`,
		`href="https://kyou.id/user/cart?utm=&lt;email&gt;"`,
		`Nendoroid Rin &lt;Touring&gt;`,
		`https://kyou.id/items/1/?x=&lt;bad&gt;`,
		`https://kyoucdn.id/items/rin.jpg.webp?x=&lt;bad&gt;`,
	}) {
		t.Fatalf("expected browser cart with button, got %q", got)
	}
	if containsAny(got, []string{`Nendoroid Rin <Touring>`, `utm=<email>`, `Ringkasan Belanja`, `Total Harga`, `Subtotal`, `Harga &amp; stok dikunci`}) {
		t.Fatalf("expected cart html to escape unsafe content, got %q", got)
	}
}

func TestBirthdayTemplatesUseFixedEmailWidth(t *testing.T) {
	for _, path := range []string{
		"../../templates/birthday/birthday1.html",
		"../../templates/preview/birthday-preview.html",
	} {
		t.Run(path, func(t *testing.T) {
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			html := string(content)
			if containsAny(html, []string{
				`<meta name="viewport"`,
				`max-width: 720px`,
				`max-width: 650px`,
				`max-width: 560px`,
				`width: 100%`,
				`width:100%`,
				`@media`,
			}) {
				t.Fatalf("expected fixed-width non-responsive template, got responsive content in %s", path)
			}
			if !containsAll(html, []string{
				`<table role="presentation" width="720"`,
				`style="width:720px;min-width:720px;`,
				`width="720"`,
			}) {
				t.Fatalf("expected fixed 720px template, got %s", path)
			}
		})
	}
}

func TestBirthdayTemplatesUseTextFooterDesign(t *testing.T) {
	for _, path := range []string{
		"../../templates/birthday/birthday1.html",
		"../../templates/preview/birthday-preview.html",
	} {
		t.Run(path, func(t *testing.T) {
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			html := string(content)
			if containsAny(html, []string{
				`static/assets/footer.jpg`,
			}) {
				t.Fatalf("expected text footer instead of footer image in %s", path)
			}
			if !containsAll(html, []string{
				`border-top: 6px solid #7a3b14`,
				`background: #efad3e`,
				`border-radius: 0 0 24px 24px`,
				`Ayo #RayakanHobimu bareng Kyou!`,
				`©2014–2026 Kyou Hobby Shop / Kyou Hobby Shop`,
				`Kamu menerima email ini karena terdaftar sebagai Teman Kyou.`,
				`margin:0 0 8px;color:#5a351d;font-size:20px;font-weight:900;line-height:1.2;`,
				`margin:0 0 0px;color:#7b5c2b;font-size:16px;font-weight:500;line-height:1.4;`,
				`margin:0;color:#8a6c36;font-size:16px;font-weight:500;line-height:1.55;`,
			}) {
				t.Fatalf("expected screenshot-style text footer in %s", path)
			}
			if containsAny(html, []string{
				`Unsubscribe`,
				`Update preferences`,
				`user/notification`,
			}) {
				t.Fatalf("expected footer without unsubscribe/preferences links in %s", path)
			}
		})
	}
}

func ptrTime(value time.Time) *time.Time {
	return &value
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

func countOccurrences(value string, needle string) int {
	count := 0
	for i := 0; i+len(needle) <= len(value); i++ {
		if value[i:i+len(needle)] == needle {
			count++
		}
	}
	return count
}
