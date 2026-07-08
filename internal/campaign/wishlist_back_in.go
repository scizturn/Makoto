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

func (c WishlistBackInCampaign) BuildMergeData(user domain.User, voucherCode string, discountPercent int, items []domain.WishlistBackInItem, companion domain.WishlistBackInItem, recos []domain.WishlistBackInItem, greeting string) map[string]any {
	// The "Lengkapin koleksi" cross-sell renders only with an anchor (a purchased
	// item) AND a full set of popular recommendations in its series/category.
	hasReco := companion.ID != "" && len(recos) > 0
	recoSeries := ""
	if hasReco {
		recoSeries = wishlistBackInRecoSeries(companion, recos)
	}
	// The coupon block prints discountPercent, so it renders only when Yukari told
	// us the tier. A job serialized before the tier existed carries a code but no
	// percent; suppressing the block there is better than shouting "0%". The
	// voucher is still claimable from the item links' ?claim= parameter.
	hasVoucher := strings.TrimSpace(voucherCode) != "" && discountPercent > 0
	return map[string]any{
		"name":              user.Name,
		"first_name":        firstName(user.Name),
		"greeting":          greeting,
		"voucher_code":      voucherCode,
		"voucher_discount":  discountPercent,
		"has_voucher":       hasVoucher,
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

// renderWishlistBackInItems renders each restocked wishlist item as its own
// "Soft Cream" card, newest first, separated by a small spacer.
func renderWishlistBackInItems(items []domain.WishlistBackInItem) string {
	var builder strings.Builder
	index := 0
	for _, item := range items {
		if item.ID == "" {
			continue
		}
		index++
		if index > 1 {
			builder.WriteString(`<div style="height:10px;line-height:10px;font-size:10px;">&nbsp;</div>`)
		}
		builder.WriteString(renderWishlistBackInRow(item, index))
	}
	return builder.String()
}

// wishlistBackInGrip is the list marker: a 2×3 dot grid (drag-handle / "six on a
// dice" grip). Table-based with per-cell padding for spacing (Gmail-safe); dots
// are <div>s so border-radius holds as circles.
func wishlistBackInGrip() string {
	dot := `<td style="padding:2px;"><div style="width:5px;height:5px;border-radius:50%;background:#c9bcb0;font-size:0;line-height:0;">&nbsp;</div></td>`
	row := `<tr>` + dot + dot + `</tr>`
	return `<table role="presentation" cellpadding="0" cellspacing="0" border="0" align="center" style="margin:0 auto;border-collapse:collapse;">` + row + row + row + `</table>`
}

// renderWishlistBackInRow — "Soft Cream" index row: NN · thumb · status/disc/name · price/sub.
func renderWishlistBackInRow(item domain.WishlistBackInItem, index int) string {
	name := html.EscapeString(cleanItemName(item.Name))
	itemURL := html.EscapeString(item.URL)
	imageURL := html.EscapeString(item.ImageURL)

	imgCell := `&nbsp;`
	if imageURL != "" {
		imgCell = fmt.Sprintf(`<img src="%s" alt="" width="96" height="96" style="display:block;width:96px;height:96px;object-fit:contain;border:0;margin:0 auto;">`, imageURL)
	}

	badge := renderStatusBadge(item)
	discTag := ""
	if item.DiscountPrice > 0 && item.DiscountPrice < item.Price {
		discTag = fmt.Sprintf(`&nbsp;<span style="font-size:9.5px;font-weight:800;color:#fc4c02;vertical-align:middle;">&minus;%d%%</span>`, (item.Price-item.DiscountPrice)*100/item.Price)
	}
	priceMain, priceColor, sub, struck := wishlistBackInPrice(item)
	subHTML := ""
	if sub != "" {
		deco := ""
		if struck {
			deco = "text-decoration:line-through;"
		}
		subHTML = fmt.Sprintf(`<div style="font-size:11px;color:#a2a2a2;white-space:nowrap;margin-top:2px;%s">%s</div>`, deco, sub)
	}

	// Each item is its own white card (Gmail/Outlook safe — no flexbox); the marker
	// is a solid dot. The first (newest) item is highlighted with an orange border.
	inner := fmt.Sprintf(`<table role="presentation" width="100%%" cellpadding="0" cellspacing="0" border="0" style="width:100%%;border-collapse:collapse;">`+
		`<tr>`+
		`<td width="22" valign="middle" align="center" style="width:22px;">%s</td>`+
		`<td width="128" valign="middle" style="width:128px;padding:0 16px 0 12px;">`+
		`<table role="presentation" width="112" cellpadding="0" cellspacing="0" border="0"><tr><td width="112" height="112" align="center" valign="middle" style="width:112px;height:112px;background:#fff7f3;border-radius:12px;border:1px solid #f0e6de;">%s</td></tr></table>`+
		`</td>`+
		`<td valign="middle" style="padding:0 12px 0 0;">`+
		`<div style="margin-bottom:6px;">%s%s</div>`+
		`<div style="font-size:14px;font-weight:800;color:#2a2a2a;line-height:1.3;">%s</div>`+
		`</td>`+
		`<td valign="middle" align="right" style="white-space:nowrap;">`+
		`<div style="font-size:15px;font-weight:900;color:%s;">%s</div>%s`+
		`</td>`+
		`</tr></table>`, wishlistBackInGrip(), imgCell, badge, discTag, name, priceColor, priceMain, subHTML)

	cardBorder := "border:1px solid #f1e7df;border-bottom:4px solid #ead7ca;"
	if index == 1 {
		cardBorder = "border:2px solid #fc4c02;border-bottom:4px solid #fc4c02;" // highlight the newest item
	}
	card := fmt.Sprintf(`<table role="presentation" width="100%%" cellpadding="0" cellspacing="0" border="0" style="width:100%%;border-collapse:separate;background:#ffffff;border-radius:16px;%s"><tr><td style="padding:16px 18px;">%s</td></tr></table>`, cardBorder, inner)

	if itemURL == "" {
		return card
	}
	return fmt.Sprintf(`<a href="%s" style="display:block;color:inherit;text-decoration:none;">%s</a>`, itemURL, card)
}

// renderWishlistBackInRecoGrid — cross-sell grid: rows of 3 cards as a table.
func renderWishlistBackInRecoGrid(items []domain.WishlistBackInItem) string {
	if len(items) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(`<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="width:100%;border-collapse:collapse;">`)
	for row := 0; row*3 < len(items); row++ {
		if row > 0 {
			b.WriteString(`<tr><td colspan="3" style="height:12px;line-height:12px;font-size:12px;">&nbsp;</td></tr>`)
		}
		b.WriteString(`<tr>`)
		for col := 0; col < 3; col++ {
			pad := "0 6px 0 0"
			if col == 1 {
				pad = "0 3px"
			} else if col == 2 {
				pad = "0 0 0 6px"
			}
			b.WriteString(fmt.Sprintf(`<td width="33%%" valign="top" style="width:33.33%%;padding:%s;">`, pad))
			if idx := row*3 + col; idx < len(items) {
				b.WriteString(renderWishlistBackInCard(items[idx]))
			} else {
				b.WriteString(`&nbsp;`)
			}
			b.WriteString(`</td>`)
		}
		b.WriteString(`</tr>`)
	}
	b.WriteString(`</table>`)
	return b.String()
}

// renderWishlistBackInCard — cross-sell card as a table cell: square image, name, price.
func renderWishlistBackInCard(item domain.WishlistBackInItem) string {
	name := html.EscapeString(cleanItemName(item.Name))
	itemURL := html.EscapeString(item.URL)
	imageURL := html.EscapeString(item.ImageURL)

	imgCell := `&nbsp;`
	if imageURL != "" {
		imgCell = fmt.Sprintf(`<img src="%s" alt="" width="150" height="150" style="display:block;width:100%%;height:auto;border:0;">`, imageURL)
	}
	priceMain, priceColor, _, _ := wishlistBackInPrice(item)

	inner := fmt.Sprintf(`<table role="presentation" width="100%%" cellpadding="0" cellspacing="0" border="0" style="width:100%%;border-collapse:separate;background:#ffffff;border-radius:14px;overflow:hidden;border:1px solid #f1e7df;border-bottom:4px solid #ead7ca;">`+
		`<tr><td style="padding:0;line-height:0;font-size:0;">%s</td></tr>`+
		`<tr><td style="padding:9px 11px 12px;">`+
		`<div style="font-size:11.5px;font-weight:700;color:#2a2a2a;line-height:1.25;">%s</div>`+
		`<div style="font-size:12px;font-weight:900;color:%s;margin-top:4px;">%s</div>`+
		`</td></tr></table>`, imgCell, name, priceColor, priceMain)

	if itemURL == "" {
		return inner
	}
	return fmt.Sprintf(`<a href="%s" style="display:block;color:inherit;text-decoration:none;">%s</a>`, itemURL, inner)
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
