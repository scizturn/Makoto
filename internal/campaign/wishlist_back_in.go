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

func (c WishlistBackInCampaign) BuildMergeData(user domain.User, voucherCode string, item, companion domain.WishlistBackInItem, greeting string) map[string]any {
	return map[string]any{
		"name":              user.Name,
		"first_name":        firstName(user.Name),
		"greeting":          greeting,
		"voucher_code":      voucherCode,
		"has_voucher":       strings.TrimSpace(voucherCode) != "",
		"action_url":        actionURLWithClaim(c.ActionURL, voucherCode),
		"back_in_item_html": renderWishlistBackInItem(item, true),
		"companion_html":    renderWishlistBackInItem(companion, false),
		"companion_name":    companion.Name,
		"has_companion":     companion.ID != "",
		"closing":           c.Closing,
		"footer_html":       RenderMemberversaryFooterHTML(c.Closing),
	}
}

// renderWishlistBackInItem renders one item for the "Editorial Index" layout.
// featured=true -> a slim numbered index row (the restocked wishlist item).
// featured=false -> a compact cross-sell card (a companion to complete the set).
func renderWishlistBackInItem(item domain.WishlistBackInItem, featured bool) string {
	if item.ID == "" {
		return ""
	}
	if featured {
		return renderWishlistBackInRow(item)
	}
	return renderWishlistBackInCard(item)
}

// renderWishlistBackInRow is the editorial "INDEX" row: 01 · thumbnail · status/name · price.
func renderWishlistBackInRow(item domain.WishlistBackInItem) string {
	name := html.EscapeString(item.Name)
	itemURL := html.EscapeString(item.URL)
	imageURL := html.EscapeString(item.ImageURL)

	thumb := `<div style="width:62px;height:62px;border-radius:8px;background:#f7f7f7;"></div>`
	if imageURL != "" {
		thumb = fmt.Sprintf(`<img src="%s" alt="%s" width="62" height="62" style="display:block;width:62px;height:62px;object-fit:cover;border:0;border-radius:8px;background:#f7f7f7;">`, imageURL, name)
	}

	badgeText, badgeColor := "Ready Stock", "#2e9c5f"
	if isPreorderStatus(item.Status) {
		badgeText, badgeColor = "PO Dibuka Lagi", "#bf3901"
	}

	price := formatIDR(item.Price)
	if price == "" {
		price = "Cek harga"
	}

	content := fmt.Sprintf(`<table role="presentation" width="100%%" cellspacing="0" cellpadding="0" style="width:100%%;border-collapse:collapse;border-bottom:1px solid #f0efec;"><tr>`+
		`<td width="26" valign="middle" style="width:26px;padding:16px 0;font-size:13px;font-weight:900;color:#d1d3d4;">01</td>`+
		`<td width="62" valign="middle" style="width:62px;padding:16px 0;">%s</td>`+
		`<td valign="middle" style="padding:16px 14px;">`+
		`<span style="display:block;margin:0 0 3px;font-size:9.5px;font-weight:800;letter-spacing:0.5px;color:%s;text-transform:uppercase;">&#9679; %s</span>`+
		`<span style="display:block;font-size:14.5px;font-weight:800;color:#2a2a2a;line-height:1.25;">%s</span>`+
		`</td>`+
		`<td valign="middle" align="right" style="padding:16px 0;font-size:15px;font-weight:900;color:#fc4c02;white-space:nowrap;">%s</td>`+
		`</tr></table>`, thumb, badgeColor, badgeText, name, price)

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

func isPreorderStatus(status string) bool {
	status = strings.ToLower(strings.TrimSpace(status))
	return strings.Contains(status, "po") || strings.Contains(status, "pre") || strings.Contains(status, "order")
}

func firstName(name string) string {
	name = strings.TrimSpace(name)
	if index := strings.Index(name, " "); index > 0 {
		return name[:index]
	}
	return name
}
