package campaign

import (
	"fmt"
	"html"
	"math/rand"
	"regexp"
	"strings"
	texttemplate "text/template"
	"time"

	"github.com/kyou-id/makoto/internal/domain"
)

// wishlistBackInLeadingTags matches one or more leading bracketed tags such as
// "[Set of 6] [With Bonus] " or an empty "[]" so they can be stripped from the
// display name. The raw name is still used for logic (e.g. the [revive] tag).
var wishlistBackInLeadingTags = regexp.MustCompile(`^\s*(?:\[[^\]]*\]\s*)+`)

// cleanItemName strips leading promotional/bracket tags for display, keeping the
// real product title. Falls back to the original if stripping empties it.
func cleanItemName(name string) string {
	out := strings.TrimSpace(wishlistBackInLeadingTags.ReplaceAllString(name, ""))
	if out == "" {
		return strings.TrimSpace(name)
	}
	return out
}

type WishlistBackInCampaign struct {
	TemplateIDs []string
	Greetings   []string
	ActionURL   string
	Closing     string
	RandomIntn  func(n int) int
}

func (c WishlistBackInCampaign) SelectTemplate(now time.Time, key string) string {
	if len(c.TemplateIDs) == 0 {
		return ""
	}
	randomIntn := c.RandomIntn
	if randomIntn == nil {
		randomIntn = rand.New(rand.NewSource(templateSeed(now, key))).Intn
	}
	return c.TemplateIDs[randomIntn(len(c.TemplateIDs))]
}

func (c WishlistBackInCampaign) SelectGreeting(now time.Time, key string) string {
	if len(c.Greetings) == 0 {
		return ""
	}
	randomIntn := c.RandomIntn
	if randomIntn == nil {
		randomIntn = rand.New(rand.NewSource(templateSeed(now, key+"_greeting"))).Intn
	}
	return c.Greetings[randomIntn(len(c.Greetings))]
}

func (c WishlistBackInCampaign) RenderGreeting(tpl string, user domain.User) string {
	firstName := firstName(user.Name)
	t, err := texttemplate.New("greeting").Parse(tpl)
	if err != nil {
		return tpl
	}
	var output strings.Builder
	if err := t.Execute(&output, struct{ Name, FirstName string }{user.Name, firstName}); err != nil {
		return tpl
	}
	return output.String()
}

func (c WishlistBackInCampaign) BuildMergeData(user domain.User, voucherCode string, items []domain.WishlistBackInItem, companion domain.WishlistBackInItem, recos []domain.WishlistBackInItem, greeting string) map[string]any {
	// The "Lengkapin koleksi" cross-sell renders only with an anchor (a purchased
	// item) AND a full set of popular recommendations in its series/category.
	hasReco := companion.ID != "" && len(recos) > 0
	recoSeries := ""
	if hasReco {
		recoSeries = wishlistBackInRecoSeries(companion, recos)
	}
	return map[string]any{
		"name":              user.Name,
		"first_name":        firstName(user.Name),
		"greeting":          greeting,
		"voucher_code":      voucherCode,
		"has_voucher":       strings.TrimSpace(voucherCode) != "",
		"action_url":        actionURLWithClaim(c.ActionURL, voucherCode),
		"back_in_item_html": renderWishlistBackInItems(items),
		"item_count":        len(items),
		"reco_html":         renderWishlistBackInRecoGrid(recos),
		"reco_series":       recoSeries,
		"companion_name":    cleanItemName(companion.Name),
		"has_companion":     hasReco,
		"closing":           c.Closing,
		"footer_html":       RenderMemberversaryFooterHTML(c.Closing),
	}
}

// wishlistBackInRecoSeries picks the series label shown in "Lengkapin koleksi <x>".
func wishlistBackInRecoSeries(companion domain.WishlistBackInItem, recos []domain.WishlistBackInItem) string {
	if s := strings.TrimSpace(companion.SeriesName); s != "" {
		return s
	}
	for _, r := range recos {
		if s := strings.TrimSpace(r.SeriesName); s != "" {
			return s
		}
	}
	return "koleksimu"
}

// renderWishlistBackInItems renders the restocked wishlist items as the "Soft
// Cream" numbered index rows (01, 02, …), newest first.
func renderWishlistBackInItems(items []domain.WishlistBackInItem) string {
	var builder strings.Builder
	index := 0
	for _, item := range items {
		if item.ID == "" {
			continue
		}
		index++
		builder.WriteString(renderWishlistBackInRow(item, index))
	}
	return builder.String()
}

// renderWishlistBackInRow — "Soft Cream" index row: NN · thumb · status/disc/name · price/sub.
func renderWishlistBackInRow(item domain.WishlistBackInItem, index int) string {
	name := html.EscapeString(cleanItemName(item.Name))
	itemURL := html.EscapeString(item.URL)
	imageURL := html.EscapeString(item.ImageURL)

	imgTag := `<div style="width:80%;height:80%;"></div>`
	if imageURL != "" {
		imgTag = fmt.Sprintf(`<img src="%s" alt="" style="width:80%%;height:80%%;object-fit:contain;">`, imageURL)
	}

	badge := renderStatusBadge(item)
	discTag := ""
	if item.DiscountPrice > 0 && item.DiscountPrice < item.Price {
		discTag = fmt.Sprintf(`<span style="font-size:9.5px;font-weight:800;color:#fc4c02;">&minus;%d%%</span>`, (item.Price-item.DiscountPrice)*100/item.Price)
	}
	priceMain, priceColor, sub, struck := wishlistBackInPrice(item)
	subHTML := ""
	if sub != "" {
		deco := ""
		if struck {
			deco = "text-decoration:line-through;"
		}
		subHTML = fmt.Sprintf(`<div style="font-size:11px;color:#a2a2a2;white-space:nowrap;%s">%s</div>`, deco, sub)
	}

	row := fmt.Sprintf(`<div style="display:flex;align-items:flex-start;gap:15px;padding:16px 0;border-bottom:1px solid #f2e7df;">`+
		`<span style="font-size:13px;font-weight:900;color:#e0c7ba;width:22px;flex:none;align-self:center;text-align:center;">%02d</span>`+
		`<div style="width:62px;height:62px;border-radius:10px;background:#ffffff;display:flex;align-items:center;justify-content:center;flex:none;overflow:hidden;box-shadow:0 2px 8px rgba(0,0,0,0.05);">%s</div>`+
		`<div style="flex:1;min-width:0;padding-top:1px;">`+
		`<div style="display:flex;align-items:center;gap:8px;margin-bottom:5px;">`+
		`%s%s`+
		`</div>`+
		`<div style="font-size:14px;font-weight:800;color:#2a2a2a;line-height:1.3;">%s</div>`+
		`</div>`+
		`<div style="text-align:right;flex:none;padding-top:1px;">`+
		`<div style="font-size:15px;font-weight:900;color:%s;white-space:nowrap;">%s</div>%s`+
		`</div>`+
		`</div>`, index, imgTag, badge, discTag, name, priceColor, priceMain, subHTML)

	if itemURL == "" {
		return row
	}
	return fmt.Sprintf(`<a href="%s" style="display:block;color:inherit;text-decoration:none;">%s</a>`, itemURL, row)
}

// renderWishlistBackInRecoGrid — "Soft Cream" cross-sell: rows of 3 flex cards.
func renderWishlistBackInRecoGrid(items []domain.WishlistBackInItem) string {
	if len(items) == 0 {
		return ""
	}
	var b strings.Builder
	for row := 0; row*3 < len(items); row++ {
		if row > 0 {
			b.WriteString(`<div style="height:12px;line-height:12px;font-size:12px;">&nbsp;</div>`)
		}
		b.WriteString(`<div style="display:flex;gap:12px;">`)
		for col := 0; col < 3; col++ {
			if idx := row*3 + col; idx < len(items) {
				b.WriteString(renderWishlistBackInCard(items[idx]))
			} else {
				b.WriteString(`<div style="flex:1;"></div>`)
			}
		}
		b.WriteString(`</div>`)
	}
	return b.String()
}

// renderWishlistBackInCard — "Soft Cream" cross-sell card (flex:1): image, name, price.
func renderWishlistBackInCard(item domain.WishlistBackInItem) string {
	name := html.EscapeString(cleanItemName(item.Name))
	itemURL := html.EscapeString(item.URL)
	imageURL := html.EscapeString(item.ImageURL)

	imgTag := `<div style="width:100%;height:100%;background:#ffffff;"></div>`
	if imageURL != "" {
		imgTag = fmt.Sprintf(`<img src="%s" alt="" style="display:block;width:100%%;height:100%%;object-fit:cover;">`, imageURL)
	}
	priceMain, priceColor, _, _ := wishlistBackInPrice(item)

	inner := fmt.Sprintf(`<div style="background:#ffffff;border-radius:14px;overflow:hidden;box-shadow:0 3px 12px rgba(0,0,0,0.04);">`+
		`<div style="aspect-ratio:1;background:#ffffff;overflow:hidden;">%s</div>`+
		`<div style="padding:9px 11px 12px;">`+
		`<div style="font-size:11.5px;font-weight:700;color:#2a2a2a;line-height:1.25;max-height:29px;overflow:hidden;">%s</div>`+
		`<div style="font-size:12px;font-weight:900;color:%s;margin-top:4px;">%s</div>`+
		`</div></div>`, imgTag, name, priceColor, priceMain)

	if itemURL == "" {
		return fmt.Sprintf(`<div style="flex:1;">%s</div>`, inner)
	}
	return fmt.Sprintf(`<a href="%s" style="flex:1;display:block;color:inherit;text-decoration:none;">%s</a>`, itemURL, inner)
}

// renderStatusBadge is hanamaru's ProductStatus pill: a solid rounded chip with
// white bold text. Colors and labels mirror the site (StatusChip); Revive uses
// the shared image tag from the `[revive]` name tag.
func renderStatusBadge(item domain.WishlistBackInItem) string {
	if strings.Contains(strings.ToLower(item.Name), "[revive]") {
		return `<img src="https://kyoucdn.id/static/img/status-tags/revive.png" alt="Revive" style="height:18px;width:auto;border:0;vertical-align:middle;">`
	}
	label, bg := "Ready Stock", "#41b774"
	switch strings.ToUpper(strings.TrimSpace(item.Status)) {
	case "PO":
		label, bg = "Pre-Order", "#657996"
	case "LPO":
		label, bg = "Late Pre-Order", "#d3647a"
	case "BO", "BPO":
		label, bg = "Back Order", "#996291"
	}
	return fmt.Sprintf(`<span style="display:inline-block;padding:4px 7px;border-radius:4px;background:%s;color:#ffffff;font-size:9.5px;font-weight:800;line-height:1;white-space:nowrap;">%s</span>`, bg, html.EscapeString(label))
}

// wishlistBackInPrice returns the main price text + color, and an optional sub
// line (struck original for a discount, or "/ <full>" for a PO down payment).
func wishlistBackInPrice(item domain.WishlistBackInItem) (main, color, sub string, struck bool) {
	switch {
	case item.DownPayment > 0:
		return "DP " + formatIDR(item.DownPayment), "#fc4c02", "/ " + formatIDRNumber(item.Price), false
	case item.DiscountPrice > 0 && item.DiscountPrice < item.Price:
		return formatIDR(item.DiscountPrice), "#fc4c02", formatIDRNumber(item.Price), true
	default:
		main = formatIDR(item.Price)
		if main == "" {
			main = "Cek harga"
		}
		return main, "#2a2a2a", "", false
	}
}

// formatIDRNumber is formatIDR without the "IDR " prefix (for the struck /
// "/ full" sub-line).
func formatIDRNumber(price int) string {
	return strings.TrimPrefix(formatIDR(price), "IDR ")
}

func firstName(name string) string {
	name = strings.TrimSpace(name)
	if index := strings.Index(name, " "); index > 0 {
		return name[:index]
	}
	return name
}
