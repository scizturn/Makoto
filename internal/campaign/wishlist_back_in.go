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
	}
}

func renderWishlistBackInItem(item domain.WishlistBackInItem, featured bool) string {
	if item.ID == "" {
		return ""
	}
	name := html.EscapeString(item.Name)
	series := html.EscapeString(item.SeriesName)
	if series == "" {
		series = html.EscapeString(item.CategoryName)
	}
	imageURL := html.EscapeString(item.ImageURL)
	itemURL := html.EscapeString(item.URL)
	image := `<div style="width:190px;height:190px;background:#f3f4f6;border-radius:8px;"></div>`
	if imageURL != "" {
		image = fmt.Sprintf(`<img src="%s" alt="%s" width="190" height="190" style="display:block;width:190px;height:190px;object-fit:cover;border:0;border-radius:8px;background:#f3f4f6;">`, imageURL, name)
	}
	price := formatIDR(item.Price)
	if price == "" {
		price = "Cek harga terbaru"
	}
	badge := "Sudah kamu beli"
	if featured {
		badge = "Ready lagi"
	}
	content := fmt.Sprintf(`<table role="presentation" width="100%%" cellspacing="0" cellpadding="0" style="width:100%%;border-collapse:collapse;background:#ffffff;border:1px solid #e5e7eb;border-radius:8px;"><tr><td width="220" valign="top" style="width:220px;padding:15px;">%s</td><td valign="middle" style="padding:20px 24px 20px 4px;"><span style="display:inline-block;margin:0 0 10px;padding:6px 9px;border-radius:4px;background:#fc4c02;color:#ffffff;font-size:11px;font-weight:900;line-height:1;">%s</span><p style="margin:0 0 6px;color:#8b8b8b;font-size:11px;font-weight:800;line-height:1.3;text-transform:uppercase;">%s</p><h2 style="margin:0 0 10px;color:#2d2d2d;font-size:21px;font-weight:900;line-height:1.25;">%s</h2><p style="margin:0;color:#fc4c02;font-size:17px;font-weight:900;line-height:1.3;">%s</p></td></tr></table>`, image, badge, series, name, price)
	if itemURL == "" {
		return content
	}
	return fmt.Sprintf(`<a href="%s" style="display:block;color:inherit;text-decoration:none;">%s</a>`, itemURL, content)
}

func firstName(name string) string {
	name = strings.TrimSpace(name)
	if index := strings.Index(name, " "); index > 0 {
		return name[:index]
	}
	return name
}
