package campaign

import (
	"math/rand"
	"strings"
	texttemplate "text/template"
	"time"

	"github.com/kyou-id/makoto/internal/domain"
)

type LeftoverCartCampaign struct {
	TemplateIDs []string
	Greetings   []string
	CartURL     string
	Closing     string
	RandomIntn  func(n int) int
}

func (c LeftoverCartCampaign) SelectTemplate(now time.Time, key string) string {
	if len(c.TemplateIDs) == 0 {
		return ""
	}
	randomIntn := c.RandomIntn
	if randomIntn == nil {
		randomIntn = rand.New(rand.NewSource(templateSeed(now, key))).Intn
	}
	return c.TemplateIDs[randomIntn(len(c.TemplateIDs))]
}

func (c LeftoverCartCampaign) SelectGreeting(now time.Time, key string) string {
	if len(c.Greetings) == 0 {
		return ""
	}
	// use a different seed offset so greeting and template don't always co-vary
	randomIntn := c.RandomIntn
	if randomIntn == nil {
		randomIntn = rand.New(rand.NewSource(templateSeed(now, key+"_greeting"))).Intn
	}
	return c.Greetings[randomIntn(len(c.Greetings))]
}

func (c LeftoverCartCampaign) RenderGreeting(tpl string, user domain.User) string {
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

func (c LeftoverCartCampaign) BuildMergeData(user domain.User, cartItems []domain.WishlistItem, recoItems []domain.FYPItem, greeting string) map[string]any {
	firstName := user.Name
	if i := strings.Index(user.Name, " "); i > 0 {
		firstName = user.Name[:i]
	}

	return map[string]any{
		"name":       user.Name,
		"first_name": firstName,
		"greeting":   greeting,
		"cart_items": cartItems,
		"reco_items": recoItems,
		"cart_html":  RenderCartItemsHTML(cartItems),
		"reco_html":  RenderFYPHTML(recoItems),
		"cart_url":   c.CartURL,
		"closing":    c.Closing,
	}
}

func RenderCartItemsHTML(items []domain.WishlistItem) string {
	if len(items) == 0 {
		return `<p style="margin:0;color:#6b7280;">Keranjangmu lagi nunggu kamu balik nih!</p>`
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
			item := items[i]
			builder.WriteString(`<td width="212" valign="top" align="center" style="width:212px;padding:0;text-align:center;">`)
			builder.WriteString(renderProductCard(item.Name, item.SeriesName, item.Status, item.URL, item.ImageURL))
			builder.WriteString(`</td>`)
		}
		builder.WriteString(`</tr>`)
	}

	builder.WriteString(`</table>`)
	return builder.String()
}
