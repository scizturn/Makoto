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

func (c WinbackCampaign) BuildMergeData(user domain.User, voucherCode string, wishlist []domain.WishlistItem, historicalItems []domain.HistoricalItem, greeting string) map[string]any {
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
		// "Album Kenangan" scrapbook from real data. Reuse the existing
		// historical_html/wishlist_html merge keys so no new viewData field is
		// needed — winback templates render their own scrapbook style. The past
		// collection is a list (thumbnail + name + purchase date); the ready
		// wishlist stays a polaroid grid.
		"historical_html": RenderWinbackPastListHTML(historicalItems),
		"wishlist_html":   RenderWinbackReadyPolaroidsHTML(wishlist),
		"action_url":      actionURLWithClaim(c.ActionURL, voucherCode),
		"closing":         c.Closing,
		// Shared Kyou footer (same as discounted wishlist).
		"footer_html": RenderMemberversaryFooterHTMLWithCTA(c.Closing, "https://kyou.id/", "Explore kyou yuk!"),
	}
}

const (
	winbackReadyLimit = 12
	winbackReadyCols  = 3
)

// winbackPastLimit caps how many past orders the "past collection" list shows.
const winbackPastLimit = 3

// RenderWinbackPastListHTML renders the user's most-recent purchases (up to
// winbackPastLimit) as a vertical list: a thumbnail on the left, the item name
// and purchase date on the right ("halaman lama dari rak kamu").
func RenderWinbackPastListHTML(items []domain.HistoricalItem) string {
	cleaned := make([]domain.HistoricalItem, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item.Name) == "" && strings.TrimSpace(item.ImageURL) == "" {
			continue
		}
		cleaned = append(cleaned, item)
	}
	if len(cleaned) == 0 {
		return `<div style="font-family:'Nunito',Arial,Helvetica,sans-serif;font-style:italic;font-size:13px;color:#9a866a;text-align:center;padding:6px 0 4px;">Rak kamu masih rapih nyimpen semua koleksi lama — nunggu halaman baru dari kamu.</div>`
	}
	if len(cleaned) > winbackPastLimit {
		cleaned = cleaned[:winbackPastLimit]
	}
	var b strings.Builder
	b.WriteString(`<table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="border-collapse:collapse;">`)
	for i, item := range cleaned {
		if i > 0 {
			b.WriteString(`<tr><td style="height:10px;line-height:10px;font-size:10px;">&nbsp;</td></tr>`)
		}
		b.WriteString(`<tr><td style="padding:0;">` + winbackPastRow(item) + `</td></tr>`)
	}
	b.WriteString(`</table>`)
	return b.String()
}

// winbackPastRow renders one past-purchase list row as a nested <table>:
// thumbnail on the left, item name and purchase date stacked on the right.
func winbackPastRow(item domain.HistoricalItem) string {
	safeName := html.EscapeString(winbackListCaption(item.Name))
	safeImage := html.EscapeString(item.ImageURL)

	imgHTML := fmt.Sprintf(`<img src="%s" alt="" width="200" height="200" style="display:block;width:200px;height:200px;border:0;border-radius:10px;">`, safeImage)
	if safeImage == "" {
		imgHTML = `<div style="width:200px;height:200px;background:#efe7d3;border-radius:10px;"></div>`
	}

	dateHTML := ""
	if dateText := winbackOrderDateText(item.OrderDate); dateText != "" {
		dateHTML = fmt.Sprintf(`<div style="font-family:'Nunito',Arial,Helvetica,sans-serif;font-size:13.5px;color:#9a866a;margin-top:7px;">%s</div>`, html.EscapeString(dateText))
	}

	row := fmt.Sprintf(
		`<table role="presentation" width="100%%" cellspacing="0" cellpadding="0" style="border-collapse:collapse;background:#fffdf6;border:1px solid #ece3d0;box-shadow:1px 2px 6px rgba(74,59,42,0.12);"><tr><td width="200" valign="top" style="width:200px;padding:16px;">%s</td><td valign="middle" style="padding:16px 20px 16px 0;"><div style="font-family:'Nunito',Arial,Helvetica,sans-serif;font-weight:800;font-size:18px;color:#4a3b2a;line-height:1.35;">%s</div>%s</td></tr></table>`,
		imgHTML, safeName, dateHTML,
	)
	// Each row links to the item so the reader can jump straight to its page.
	if strings.TrimSpace(item.URL) == "" {
		return row
	}
	return fmt.Sprintf(`<a href="%s" style="text-decoration:none;color:inherit;display:block;">%s</a>`, html.EscapeString(item.URL), row)
}

// winbackListCaption cleans an item name for the wider list layout: it strips
// leading bracket tags (e.g. "[PSA 10] [with Bonus] Foo" → "Foo") and truncates
// what remains. The polaroid grid uses the tighter winbackCaption.
func winbackListCaption(name string) string {
	name = stripBracketPrefixes(name)
	runes := []rune(name)
	if len(runes) > 60 {
		return strings.TrimSpace(string(runes[:60])) + "…"
	}
	return name
}

// stripBracketPrefixes removes one or more leading "[...]" tags (and the
// whitespace around them) from an item name, so store labels like
// "[Exclusive Sale] [with Bonus] Figure ..." render as just "Figure ...".
func stripBracketPrefixes(name string) string {
	name = strings.TrimSpace(name)
	for strings.HasPrefix(name, "[") {
		end := strings.IndexByte(name, ']')
		if end < 0 {
			break
		}
		name = strings.TrimSpace(name[end+1:])
	}
	return name
}

// winbackOrderDateText formats a purchase date in Indonesian, e.g.
// "Dibeli 1 Mei 2026". Returns "" for a zero date.
func winbackOrderDateText(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return fmt.Sprintf("Dibeli %d %s %d", t.Day(), idMonths[t.Month()], t.Year())
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
	var b strings.Builder
	b.WriteString(`<table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="border-collapse:collapse;">`)
	for row := 0; row < len(items); row += winbackReadyCols {
		if row > 0 {
			b.WriteString(`<tr><td colspan="3" style="height:16px;line-height:16px;font-size:16px;">&nbsp;</td></tr>`)
		}
		b.WriteString(`<tr>`)
		end := row + winbackReadyCols
		if end > len(items) {
			end = len(items)
		}
		for i := row; i < end; i++ {
			caption := winbackCaption(items[i].Name)
			if price := formatIDR(items[i].Price); price != "" {
				caption = caption + " · " + price
			}
			card := winbackPolaroidCard(items[i].ImageURL, caption, "#4a3b2a", items[i].IsWishlisted, items[i].URL)
			b.WriteString(`<td width="33.33%" valign="top" align="center" style="padding:0 6px;">` + card + `</td>`)
		}
		for i := end; i < row+winbackReadyCols; i++ {
			b.WriteString(`<td width="33.33%">&nbsp;</td>`)
		}
		b.WriteString(`</tr>`)
	}
	b.WriteString(`</table>`)
	return b.String()
}

// winbackPolaroidCard renders one scrapbook polaroid as a nested <table>. It uses
// no position:absolute (Gmail/Outlook strip it) and no transform:rotate (Outlook
// ignores it, and Apple Mail would honour it — producing inconsistent tilt
// between recipients). The card is designed flat so the preview matches what
// every email client actually shows.
func winbackPolaroidCard(imageURL, caption, captionColor string, highlight bool, linkURL string) string {
	safeCaption := html.EscapeString(caption)
	safeImage := html.EscapeString(imageURL)

	// alt is intentionally empty: the caption <div> right below already names the
	// item, so a meaningful alt would duplicate that text (visible when images are
	// blocked or when the email text is extracted).
	imgHTML := fmt.Sprintf(`<img src="%s" alt="" width="190" style="display:block;width:100%%;max-width:190px;height:auto;border:0;">`, safeImage)
	if safeImage == "" {
		imgHTML = `<div style="height:140px;background:#efe7d3;"></div>`
	}

	// The user's own wishlist items get a 2px orange border to stand apart from
	// the popular-ready fill items.
	borderStyle := "border:1px solid #ece3d0;"
	if highlight {
		borderStyle = "border:2px solid #fc4c02;"
	}

	card := fmt.Sprintf(
		`<table role="presentation" width="190" cellspacing="0" cellpadding="0" style="width:190px;margin:0 auto;border-collapse:collapse;background:#fffdf6;box-shadow:2px 4px 9px rgba(74,59,42,0.2);%s"><tr><td style="padding:8px 8px 11px;"><div style="background:#efe7d3;font-size:0;line-height:0;">%s</div><div style="font-family:'Nunito',Arial,Helvetica,sans-serif;font-weight:800;font-size:14px;color:%s;text-align:center;padding:8px 3px 0;line-height:1.25;">%s</div></td></tr></table>`,
		borderStyle, imgHTML, captionColor, safeCaption,
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
