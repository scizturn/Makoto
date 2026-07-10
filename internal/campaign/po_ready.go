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

// PoReadyCampaign renders the "item PO di wishlist kamu udah ready" email: pure
// availability news for items the user wishlisted, whose pre-order stock the
// warehouse just converted to ready. No voucher, no balance — one CTA back to the
// user's wishlist.
type PoReadyCampaign struct {
	TemplateIDs []string
	Subjects    []string
	Greetings   []string
	WishlistURL string
	Closing     string
	RandomIntn  func(n int) int
}

// poReadyItemLimit is a defensive cap; Yukari's reader already sends at most 5.
const poReadyItemLimit = 5

func (c PoReadyCampaign) SelectTemplate(now time.Time, key string) string {
	if len(c.TemplateIDs) == 0 {
		return ""
	}
	randomIntn := c.RandomIntn
	if randomIntn == nil {
		randomIntn = rand.New(rand.NewSource(templateSeed(now, key))).Intn
	}
	return c.TemplateIDs[randomIntn(len(c.TemplateIDs))]
}

func (c PoReadyCampaign) SelectSubject(now time.Time, key string) string {
	if len(c.Subjects) == 0 {
		return ""
	}
	randomIntn := c.RandomIntn
	if randomIntn == nil {
		randomIntn = rand.New(rand.NewSource(templateSeed(now, key+"_subject"))).Intn
	}
	return c.Subjects[randomIntn(len(c.Subjects))]
}

func (c PoReadyCampaign) RenderSubject(tpl string, user domain.User) (string, error) {
	t, err := texttemplate.New("subject").Parse(tpl)
	if err != nil {
		return tpl, err
	}
	var buf strings.Builder
	if err := t.Execute(&buf, poReadyNameData(user)); err != nil {
		return tpl, err
	}
	return buf.String(), nil
}

func (c PoReadyCampaign) SelectGreeting(now time.Time, key string) string {
	if len(c.Greetings) == 0 {
		return ""
	}
	randomIntn := c.RandomIntn
	if randomIntn == nil {
		randomIntn = rand.New(rand.NewSource(templateSeed(now, key+"_greeting"))).Intn
	}
	return c.Greetings[randomIntn(len(c.Greetings))]
}

func (c PoReadyCampaign) RenderGreeting(tpl string, user domain.User) string {
	t, err := texttemplate.New("greeting").Parse(tpl)
	if err != nil {
		return tpl
	}
	var buf strings.Builder
	if err := t.Execute(&buf, poReadyNameData(user)); err != nil {
		return tpl
	}
	return buf.String()
}

func (c PoReadyCampaign) BuildMergeData(user domain.User, items []domain.PoReadyItem, greeting string) map[string]any {
	firstName := user.Name
	if i := strings.Index(user.Name, " "); i > 0 {
		firstName = user.Name[:i]
	}
	wishlistURL := c.WishlistURL
	if strings.TrimSpace(wishlistURL) == "" {
		wishlistURL = "https://kyou.id/user/wishlist"
	}
	// Truncate before counting, so item_count never promises more cards than the
	// grid renders.
	if len(items) > poReadyItemLimit {
		items = items[:poReadyItemLimit]
	}
	return map[string]any{
		"name":        user.Name,
		"first_name":  firstName,
		"greeting":    greeting,
		"items_html":  RenderPoReadyItemsHTML(items, wishlistURL),
		"item_count":  strconv.Itoa(len(items)),
		"action_url":  wishlistURL,
		"closing":     c.Closing,
		"footer_html": RenderMemberversaryFooterHTMLWithCTA(c.Closing, "https://kyou.id/", "Explore kyou yuk!"),
	}
}

func poReadyNameData(user domain.User) struct{ Name, FirstName string } {
	firstName := user.Name
	if i := strings.Index(user.Name, " "); i > 0 {
		firstName = user.Name[:i]
	}
	return struct{ Name, FirstName string }{user.Name, firstName}
}

// RenderPoReadyItemsHTML renders the readied items as a two-column grid.
func RenderPoReadyItemsHTML(items []domain.PoReadyItem, fallbackURL string) string {
	if len(items) == 0 {
		return `<p style="margin:0;color:#6b7280;">Ada barang wishlist kamu yang udah ready — cek langsung di wishlist-mu.</p>`
	}
	if len(items) > poReadyItemLimit {
		items = items[:poReadyItemLimit]
	}

	var builder strings.Builder
	builder.WriteString(`<table role="presentation" width="636" cellspacing="0" cellpadding="0" align="center" style="width:636px;border-collapse:collapse;margin:0 auto;">`)
	for rowStart := 0; rowStart < len(items); rowStart += 2 {
		rowEnd := rowStart + 2
		if rowEnd > len(items) {
			rowEnd = len(items)
		}
		if rowStart > 0 {
			builder.WriteString(`<tr><td colspan="2" style="height:14px;line-height:14px;font-size:14px;">&nbsp;</td></tr>`)
		}
		builder.WriteString(`<tr>`)
		for i := rowStart; i < rowEnd; i++ {
			builder.WriteString(`<td width="318" valign="top" align="center" style="width:318px;padding:0 7px;text-align:center;">`)
			builder.WriteString(renderPoReadyItemCard(items[i], fallbackURL))
			builder.WriteString(`</td>`)
		}
		for i := rowEnd; i < rowStart+2; i++ {
			builder.WriteString(`<td width="318" valign="top" align="center" style="width:318px;padding:0 7px;text-align:center;">&nbsp;</td>`)
		}
		builder.WriteString(`</tr>`)
	}
	builder.WriteString(`</table>`)
	return builder.String()
}

func renderPoReadyItemCard(item domain.PoReadyItem, fallbackURL string) string {
	safeName := html.EscapeString(item.Name)
	safeURL := html.EscapeString(itemURLOrFallback(item.URL, fallbackURL))
	safeImageURL := html.EscapeString(item.ImageURL)

	imgHTML := fmt.Sprintf(
		`<img src="%s" alt="%s" width="120" height="120" style="display:block;width:120px;height:120px;object-fit:cover;background:#f3f4f6;border:0;border-radius:8px;">`,
		safeImageURL, safeName,
	)
	if safeImageURL == "" {
		imgHTML = `<div style="width:120px;height:120px;background:#f3f4f6;border-radius:8px;"></div>`
	}

	// A discounted item shows the original struck through next to the live price.
	priceHTML := ""
	if item.DiscountPrice > 0 && item.DiscountPrice < item.Price {
		priceHTML = fmt.Sprintf(
			`<p style="margin:6px 0 0;line-height:1.2;"><span style="color:#2d2d2d;font-size:14px;font-weight:900;">%s</span> <span style="color:#9ca3af;font-size:12px;font-weight:700;text-decoration:line-through;">%s</span></p>`,
			formatIDR(item.DiscountPrice), formatIDR(item.Price),
		)
	} else if price := formatIDR(item.Price); price != "" {
		priceHTML = fmt.Sprintf(`<p style="margin:6px 0 0;color:#2d2d2d;font-size:14px;font-weight:900;line-height:1.2;">%s</p>`, price)
	}

	inner := fmt.Sprintf(
		`<table role="presentation" width="304" cellspacing="0" cellpadding="0" style="width:304px;border-collapse:separate;border:1px solid #e8e1d6;border-radius:10px;background:#ffffff;"><tr><td width="136" valign="middle" align="center" style="width:136px;padding:12px;">%s</td><td valign="middle" align="left" style="padding:12px 14px 12px 0;text-align:left;"><span style="display:inline-block;margin-bottom:6px;padding:3px 8px;border-radius:999px;background:#e8f5e9;color:#2e7d32;font-size:9px;font-weight:900;letter-spacing:.5px;text-transform:uppercase;">READY</span><p style="margin:0;color:#2d2d2d;font-size:13px;font-weight:800;line-height:1.3;">%s</p>%s</td></tr></table>`,
		imgHTML, safeName, priceHTML,
	)

	if safeURL == "" {
		return inner
	}
	return fmt.Sprintf(`<a href="%s" style="display:block;width:304px;margin:auto;color:inherit;text-decoration:none;">%s</a>`, safeURL, inner)
}
