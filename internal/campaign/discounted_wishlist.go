package campaign

import (
	"fmt"
	"html"
	"math/rand"
	"strconv"
	"strings"
	texttemplate "text/template"
	"time"

	"github.com/kyou-id/makoto/internal/domain"
)

type DiscountedWishlistCampaign struct {
	TemplateIDs []string
	Greetings   []string
	WishlistURL string
	Closing     string
	RandomIntn  func(n int) int
}

func (c DiscountedWishlistCampaign) SelectTemplate(now time.Time, key string) string {
	if len(c.TemplateIDs) == 0 {
		return ""
	}
	randomIntn := c.RandomIntn
	if randomIntn == nil {
		randomIntn = rand.New(rand.NewSource(templateSeed(now, key))).Intn
	}
	return c.TemplateIDs[randomIntn(len(c.TemplateIDs))]
}

func (c DiscountedWishlistCampaign) SelectGreeting(now time.Time, key string) string {
	if len(c.Greetings) == 0 {
		return ""
	}
	randomIntn := c.RandomIntn
	if randomIntn == nil {
		randomIntn = rand.New(rand.NewSource(templateSeed(now, key+"_greeting"))).Intn
	}
	return c.Greetings[randomIntn(len(c.Greetings))]
}

func (c DiscountedWishlistCampaign) RenderGreeting(tpl string, user domain.User) string {
	firstName := user.Name
	if i := strings.Index(user.Name, " "); i > 0 {
		firstName = user.Name[:i]
	}
	t, err := texttemplate.New("greeting").Parse(tpl)
	if err != nil {
		return tpl
	}
	var buf strings.Builder
	if err := t.Execute(&buf, struct{ Name, FirstName string }{user.Name, firstName}); err != nil {
		return tpl
	}
	return buf.String()
}

func (c DiscountedWishlistCampaign) BuildMergeData(user domain.User, items []domain.DiscountedWishlistItem, greeting string) map[string]any {
	firstName := user.Name
	if i := strings.Index(user.Name, " "); i > 0 {
		firstName = user.Name[:i]
	}

	var wishlisted, fill []domain.DiscountedWishlistItem
	for _, item := range items {
		if item.IsWishlisted {
			wishlisted = append(wishlisted, item)
		} else {
			fill = append(fill, item)
		}
	}

	return map[string]any{
		"name":         user.Name,
		"first_name":   firstName,
		"greeting":     greeting,
		"wishlist_html": RenderDiscountedItemsHTML(wishlisted),
		"fill_html":    RenderDiscountedItemsHTML(fill),
		"wishlist_url": c.WishlistURL,
		"closing":      c.Closing,
	}
}

func RenderDiscountedItemsHTML(items []domain.DiscountedWishlistItem) string {
	if len(items) == 0 {
		return `<p style="margin:0;color:#6b7280;">Nantikan promo menarik berikutnya!</p>`
	}

	var builder strings.Builder
	builder.WriteString(`<table role="presentation" width="636" cellspacing="0" cellpadding="0" align="center" style="width:636px;border-collapse:collapse;margin:0 auto;">`)

	for row := 0; row < 2; row++ {
		start := row * 3
		if start >= len(items) {
			break
		}
		end := start + 3
		if end > len(items) {
			end = len(items)
		}
		if row > 0 {
			builder.WriteString(`<tr><td colspan="3" style="height:12px;line-height:12px;font-size:12px;">&nbsp;</td></tr>`)
		}
		builder.WriteString(`<tr>`)
		for i := start; i < end; i++ {
			builder.WriteString(`<td width="212" valign="top" align="center" style="width:212px;padding:0;text-align:center;">`)
			builder.WriteString(renderDiscountedItemCard(items[i]))
			builder.WriteString(`</td>`)
		}
		builder.WriteString(`</tr>`)
	}

	builder.WriteString(`</table>`)
	return builder.String()
}

func renderDiscountedItemCard(item domain.DiscountedWishlistItem) string {
	mainName, version := abbreviateCardTitle(item.Name, item.SeriesName)
	safeName := html.EscapeString(mainName)
	safeVersion := html.EscapeString(version)
	safeSeries := html.EscapeString(displayManufacturerOrFallback(item.SeriesName, "Koleksi"))
	safeURL := html.EscapeString(item.URL)
	safeImageURL := html.EscapeString(item.ImageURL)
	safeDiscountName := html.EscapeString(item.DiscountName)

	imgHTML := fmt.Sprintf(
		`<img src="%s" alt="%s" width="180" height="180" style="display:block;width:180px;height:180px;object-fit:cover;background:#f3f4f6;border:0;">`,
		safeImageURL, safeName,
	)
	if safeImageURL == "" {
		imgHTML = `<div style="width:180px;height:180px;background:#f3f4f6;"></div>`
	}

	versionP := ""
	if safeVersion != "" {
		versionP = fmt.Sprintf(
			`<p style="margin:0 0 1px;color:#374151;font-size:10px;font-weight:600;line-height:1.3;">%s</p>`,
			safeVersion,
		)
	}

	originalPriceHTML := ""
	if item.OriginalPrice > 0 && item.OriginalPrice != item.DiscountPrice {
		originalPriceHTML = fmt.Sprintf(
			`<p style="margin:0 0 1px;color:#9ca3af;font-size:9px;font-weight:500;text-decoration:line-through;">%s</p>`,
			formatIDR(item.OriginalPrice),
		)
	}

	pctBadge := ""
	if item.OriginalPrice > 0 && item.DiscountPrice > 0 && item.OriginalPrice > item.DiscountPrice {
		pct := (item.OriginalPrice - item.DiscountPrice) * 100 / item.OriginalPrice
		pctBadge = fmt.Sprintf(
			`<span style="display:inline-block;margin-left:4px;padding:1px 4px;background:#dc2626;border-radius:3px;color:#ffffff;font-size:9px;font-weight:800;">-%d%%</span>`,
			pct,
		)
	}

	discountPriceHTML := ""
	if item.DiscountPrice > 0 {
		discountPriceHTML = fmt.Sprintf(
			`<p style="margin:0 0 2px;color:#dc2626;font-size:11px;font-weight:900;line-height:1.3;">%s%s</p>`,
			formatIDR(item.DiscountPrice), pctBadge,
		)
	}

	discountBadge := ""
	if safeDiscountName != "" {
		discountBadge = fmt.Sprintf(
			`<span style="display:inline-block;padding:1px 5px;background:#7c3aed;border-radius:3px;color:#ffffff;font-size:9px;font-weight:800;">%s</span>`,
			safeDiscountName,
		)
	}

	borderStyle := "1px solid #9ca3af"
	wishlistBadge := ""
	if item.IsWishlisted {
		borderStyle = "2px solid #f59e0b"
		wishlistBadge = `<div style="margin:4px 0 0;text-align:center;"><span style="display:inline-block;padding:2px 8px;background:#fef3c7;border:1px solid #f59e0b;border-radius:3px;color:#92400e;font-size:9px;font-weight:800;">★ Wishlist Kamu</span></div>`
	}

	inner := fmt.Sprintf(
		`<div style="overflow:hidden;width:180px;border:%s;border-radius:8px;background:url('https://kyoucdn.id/static/assets/item%%20bg1.jpg') center/cover no-repeat;">%s<div style="padding:8px 10px 10px;text-align:left;"><p style="margin:0 0 2px;color:#2f2b28;font-size:10px;font-weight:400;line-height:1.2;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;">%s</p><p style="margin:0 0 2px;color:#0f172a;font-size:12px;font-weight:900;line-height:1.3;white-space:normal;word-break:break-word;">%s</p>%s%s%s%s%s</div></div>`,
		borderStyle, imgHTML,
		safeSeries, safeName, versionP,
		originalPriceHTML, discountPriceHTML, discountBadge, wishlistBadge,
	)

	if safeURL == "" {
		return inner
	}
	return fmt.Sprintf(
		`<a href="%s" style="display:block;width:180px;margin:auto;color:inherit;text-decoration:none;">%s</a>`,
		safeURL, inner,
	)
}

func formatIDR(price int) string {
	if price <= 0 {
		return ""
	}
	s := strconv.Itoa(price)
	var parts []string
	for len(s) > 3 {
		parts = append([]string{s[len(s)-3:]}, parts...)
		s = s[:len(s)-3]
	}
	parts = append([]string{s}, parts...)
	return "Rp " + strings.Join(parts, ".")
}
