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

type WinbackCampaign struct {
	TemplateIDs []string
	Subjects    []string
	Greetings   []string
	ActionURL   string
	Closing     string
	RandomIntn  func(n int) int
}

func (c WinbackCampaign) SelectTemplate(now time.Time, key string) string {
	if len(c.TemplateIDs) == 0 {
		return ""
	}
	randomIntn := c.RandomIntn
	if randomIntn == nil {
		randomIntn = rand.New(rand.NewSource(templateSeed(now, key))).Intn
	}
	return c.TemplateIDs[randomIntn(len(c.TemplateIDs))]
}

func (c WinbackCampaign) SelectGreeting(now time.Time, key string) string {
	if len(c.Greetings) == 0 {
		return ""
	}
	randomIntn := c.RandomIntn
	if randomIntn == nil {
		randomIntn = rand.New(rand.NewSource(templateSeed(now, key+"_greeting"))).Intn
	}
	return c.Greetings[randomIntn(len(c.Greetings))]
}

func (c WinbackCampaign) SelectSubject(now time.Time, key string) string {
	if len(c.Subjects) == 0 {
		return ""
	}
	randomIntn := c.RandomIntn
	if randomIntn == nil {
		randomIntn = rand.New(rand.NewSource(templateSeed(now, key))).Intn
	}
	return c.Subjects[randomIntn(len(c.Subjects))]
}

func (c WinbackCampaign) RenderSubject(tpl string, user domain.User) (string, error) {
	firstName := user.Name
	if i := strings.Index(user.Name, " "); i > 0 {
		firstName = user.Name[:i]
	}
	t, err := texttemplate.New("subject").Parse(tpl)
	if err != nil {
		return "", err
	}
	var buf strings.Builder
	if err := t.Execute(&buf, struct{ Name, FirstName string }{user.Name, firstName}); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (c WinbackCampaign) RenderGreeting(tpl string, user domain.User) string {
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

func (c WinbackCampaign) BuildMergeData(user domain.User, voucherCode string, wishlist []domain.WishlistItem, historicalItem domain.HistoricalItem, greeting string) map[string]any {
	firstName := user.Name
	if i := strings.Index(user.Name, " "); i > 0 {
		firstName = user.Name[:i]
	}
	return map[string]any{
		"name":         user.Name,
		"first_name":   firstName,
		"greeting":     greeting,
		"voucher_code": voucherCode,
		"has_voucher":  strings.TrimSpace(voucherCode) != "",
		// "Album Kenangan" scrapbook polaroids from real data. Reuse the existing
		// historical_html/wishlist_html merge keys so no new viewData field is
		// needed — winback templates render their own scrapbook style.
		"historical_html": RenderWinbackPastPolaroidsHTML(historicalItem),
		"wishlist_html":   RenderWinbackReadyPolaroidsHTML(wishlist),
		"action_url":      actionURLWithClaim(c.ActionURL, voucherCode),
		"closing":         c.Closing,
		// Shared Kyou footer (same as discounted wishlist).
		"footer_html": RenderMemberversaryFooterHTMLWithCTA(c.Closing, "https://kyou.id/", "Explore kyou yuk!"),
	}
}

const winbackReadyLimit = 2

// RenderWinbackPastPolaroidsHTML renders the user's past purchase as a single
// centered scrapbook polaroid ("halaman lama dari rak kamu").
func RenderWinbackPastPolaroidsHTML(item domain.HistoricalItem) string {
	if strings.TrimSpace(item.Name) == "" && strings.TrimSpace(item.ImageURL) == "" {
		return `<div style="font-family:'Nunito',Arial,Helvetica,sans-serif;font-style:italic;font-size:13px;color:#9a866a;text-align:center;padding:6px 0 4px;">Rak kamu masih rapih nyimpen semua koleksi lama — nunggu halaman baru dari kamu.</div>`
	}
	caption := winbackCaption(item.Name)
	if !item.OrderDate.IsZero() {
		caption = caption + " · " + item.OrderDate.Format("Jan '06")
	}
	card := winbackPolaroidCard(item.ImageURL, caption, "#4a3b2a", "")
	return `<table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="border-collapse:collapse;"><tr><td align="center" style="padding:0;"><table role="presentation" cellspacing="0" cellpadding="0" style="border-collapse:collapse;"><tr><td width="250" style="width:250px;padding:0;">` + card + `</td></tr></table></td></tr></table>`
}

// RenderWinbackReadyPolaroidsHTML renders up to two wishlist items as
// polaroids ("wishlist kamu, ready stock").
func RenderWinbackReadyPolaroidsHTML(items []domain.WishlistItem) string {
	if len(items) == 0 {
		return `<div style="font-family:'Nunito',Arial,Helvetica,sans-serif;font-style:italic;font-size:13px;color:#9a866a;text-align:center;padding:6px 0 4px;">Wishlist kamu lagi kosong — mampir yuk, banyak rilisan baru yang nunggu.</div>`
	}
	if len(items) > winbackReadyLimit {
		items = items[:winbackReadyLimit]
	}
	var cells strings.Builder
	for _, item := range items {
		caption := winbackCaption(item.Name)
		if price := formatIDR(item.Price); price != "" {
			caption = caption + " · " + price
		}
		card := winbackPolaroidCard(item.ImageURL, caption, "#4a3b2a", item.URL)
		cells.WriteString(`<td width="50%" valign="top" align="center" style="padding:0 8px;">` + card + `</td>`)
	}
	if len(items) == 1 {
		cells.WriteString(`<td width="50%">&nbsp;</td>`)
	}
	return `<table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="border-collapse:collapse;"><tr>` + cells.String() + `</tr></table>`
}

// winbackPolaroidCard renders one scrapbook polaroid as a nested <table>. It uses
// no position:absolute (Gmail/Outlook strip it) and no transform:rotate (Outlook
// ignores it, and Apple Mail would honour it — producing inconsistent tilt
// between recipients). The card is designed flat so the preview matches what
// every email client actually shows.
func winbackPolaroidCard(imageURL, caption, captionColor, linkURL string) string {
	safeCaption := html.EscapeString(caption)
	safeImage := html.EscapeString(imageURL)

	// alt is intentionally empty: the caption <div> right below already names the
	// item, so a meaningful alt would duplicate that text (visible when images are
	// blocked or when the email text is extracted).
	imgHTML := fmt.Sprintf(`<img src="%s" alt="" width="210" style="display:block;width:100%%;max-width:210px;height:auto;border:0;">`, safeImage)
	if safeImage == "" {
		imgHTML = `<div style="height:150px;background:#efe7d3;"></div>`
	}

	card := fmt.Sprintf(
		`<table role="presentation" width="210" cellspacing="0" cellpadding="0" style="width:210px;margin:0 auto;border-collapse:collapse;background:#fffdf6;box-shadow:2px 4px 9px rgba(74,59,42,0.2);"><tr><td style="padding:9px 9px 12px;"><div style="background:#efe7d3;font-size:0;line-height:0;">%s</div><div style="font-family:'Nunito',Arial,Helvetica,sans-serif;font-weight:800;font-size:15px;color:%s;text-align:center;padding:9px 4px 0;line-height:1.25;">%s</div></td></tr></table>`,
		imgHTML, captionColor, safeCaption,
	)
	if strings.TrimSpace(linkURL) == "" {
		return card
	}
	return fmt.Sprintf(`<a href="%s" style="text-decoration:none;color:inherit;display:block;">%s</a>`, html.EscapeString(linkURL), card)
}

func winbackCaption(name string) string {
	name = strings.TrimSpace(name)
	runes := []rune(name)
	if len(runes) > 24 {
		return strings.TrimSpace(string(runes[:24])) + "…"
	}
	return name
}
