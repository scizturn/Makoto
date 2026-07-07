package campaign

import (
	"fmt"
	"html"
	"math/rand"
	"strings"
	texttemplate "text/template"
	"time"

	"github.com/kyou-id/makoto/internal/domain"
)

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
	// The "Gas, nemenin..." section renders only with an anchor (a purchased
	// item) AND a full set of popular recommendations in its series/category.
	hasReco := companion.ID != "" && len(recos) > 0
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
		"companion_name":    companion.Name,
		"has_companion":     hasReco,
		"closing":           c.Closing,
		"footer_html":       RenderMemberversaryFooterHTML(c.Closing),
	}
}

// renderWishlistBackInRecoGrid renders the 6 cross-sell recommendations as a
// 3-column grid (2 rows) of compact cards.
func renderWishlistBackInRecoGrid(items []domain.WishlistBackInItem) string {
	if len(items) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(`<table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="width:100%;border-collapse:collapse;">`)
	for row := 0; row*3 < len(items); row++ {
		if row > 0 {
			b.WriteString(`<tr><td colspan="3" style="height:18px;line-height:18px;font-size:18px;">&nbsp;</td></tr>`)
		}
		b.WriteString(`<tr>`)
		for col := 0; col < 3; col++ {
			b.WriteString(`<td width="33%" valign="top" align="center" style="width:33.33%;padding:0 6px;">`)
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

// renderWishlistBackInItems renders the user's restocked wishlist items as the
// editorial "INDEX" list: one numbered row per item (01, 02, …), newest first.
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

// renderWishlistBackInRow is the editorial "INDEX" row: NN · thumbnail · status/name · price.
func renderWishlistBackInRow(item domain.WishlistBackInItem, index int) string {
	name := html.EscapeString(item.Name)
	itemURL := html.EscapeString(item.URL)
	imageURL := html.EscapeString(item.ImageURL)

	thumb := `<div style="width:62px;height:62px;border-radius:8px;background:#f7f7f7;"></div>`
	if imageURL != "" {
		thumb = fmt.Sprintf(`<img src="%s" alt="%s" width="62" height="62" style="display:block;width:62px;height:62px;object-fit:cover;border:0;border-radius:8px;background:#f7f7f7;">`, imageURL, name)
	}

	badge := renderStatusBadge(item)

	price := formatIDR(item.Price)
	if price == "" {
		price = "Cek harga"
	}

	content := fmt.Sprintf(`<table role="presentation" width="100%%" cellspacing="0" cellpadding="0" style="width:100%%;border-collapse:collapse;border-bottom:1px solid #f0efec;"><tr>`+
		`<td width="26" valign="middle" style="width:26px;padding:16px 0;font-size:13px;font-weight:900;color:#d1d3d4;">%02d</td>`+
		`<td width="62" valign="middle" style="width:62px;padding:16px 0;">%s</td>`+
		`<td valign="middle" style="padding:16px 14px;">`+
		`<span style="display:block;margin:0 0 5px;">%s</span>`+
		`<span style="display:block;font-size:14.5px;font-weight:800;color:#2a2a2a;line-height:1.25;">%s</span>`+
		`</td>`+
		`<td valign="middle" align="right" style="padding:16px 0;font-size:15px;font-weight:900;color:#fc4c02;white-space:nowrap;">%s</td>`+
		`</tr></table>`, index, thumb, badge, name, price)

	if itemURL == "" {
		return content
	}
	return fmt.Sprintf(`<a href="%s" style="display:block;color:inherit;text-decoration:none;">%s</a>`, itemURL, content)
}

// renderWishlistBackInCard is the editorial "CROSS-SELL" card: square image, name, price.
func renderWishlistBackInCard(item domain.WishlistBackInItem) string {
	name := html.EscapeString(item.Name)
	itemURL := html.EscapeString(item.URL)
	imageURL := html.EscapeString(item.ImageURL)

	image := `<div style="width:150px;height:150px;border-radius:8px;background:#f7f7f7;"></div>`
	if imageURL != "" {
		image = fmt.Sprintf(`<img src="%s" alt="%s" width="150" height="150" style="display:block;width:150px;height:150px;object-fit:contain;border:0;border-radius:8px;background:#f7f7f7;">`, imageURL, name)
	}

	price := formatIDR(item.Price)
	if price == "" {
		price = "Cek harga"
	}

	content := fmt.Sprintf(`<table role="presentation" cellspacing="0" cellpadding="0" style="border-collapse:collapse;"><tr><td width="150" valign="top" style="width:150px;">`+
		`%s`+
		`<div style="margin:9px 0 0;font-size:12px;font-weight:700;color:#2a2a2a;line-height:1.3;">%s</div>`+
		`<div style="margin:3px 0 0;font-size:12.5px;font-weight:900;color:#fc4c02;">%s</div>`+
		`</td></tr></table>`, image, name, price)

	if itemURL == "" {
		return content
	}
	return fmt.Sprintf(`<a href="%s" style="display:inline-block;color:inherit;text-decoration:none;">%s</a>`, itemURL, content)
}

// renderStatusBadge mirrors hanamaru's ProductStatus badge (StatusChip / the
// ProductThumbnail status tags): same labels and colors per status. Revive uses
// the shared image tag, matching the site's `[revive]` name-tag handling.
func renderStatusBadge(item domain.WishlistBackInItem) string {
	if strings.Contains(strings.ToLower(item.Name), "[revive]") {
		return `<img src="https://kyoucdn.id/static/img/status-tags/revive.png" alt="Revive" width="68" height="20" style="display:inline-block;height:20px;width:auto;border:0;vertical-align:middle;">`
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
	return fmt.Sprintf(`<span style="display:inline-block;padding:3px 7px;border-radius:4px;background:%s;color:#ffffff;font-size:10px;font-weight:800;line-height:1.4;white-space:nowrap;">%s</span>`, bg, html.EscapeString(label))
}

func firstName(name string) string {
	name = strings.TrimSpace(name)
	if index := strings.Index(name, " "); index > 0 {
		return name[:index]
	}
	return name
}
