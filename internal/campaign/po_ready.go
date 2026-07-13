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

// poReadyMonthName are the full Indonesian month names for the send-date stamp, so
// it reads naturally (e.g. "11 Juli").
var poReadyMonthName = [...]string{"Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"}

func formatPoReadyBlastDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return fmt.Sprintf("%d %s %d", t.Day(), poReadyMonthName[int(t.Month())-1], t.Year())
}

func (c PoReadyCampaign) BuildMergeData(user domain.User, items []domain.PoReadyItem, greeting string, blastDate time.Time) map[string]any {
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
		"blast_date":  formatPoReadyBlastDate(blastDate),
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

// RenderPoReadyItemsHTML renders the readied items as full-width "manifest" rows,
// stacked one per line (newest first), each showing the Pre-Order → Ready Stock
// transition — the story of this campaign.
// An empty item list renders nothing: the worker skips such a job as
// no_ready_items before it ever reaches a template, so there is no empty-state
// copy to write.
func RenderPoReadyItemsHTML(items []domain.PoReadyItem, fallbackURL string) string {
	if len(items) > poReadyItemLimit {
		items = items[:poReadyItemLimit]
	}

	var builder strings.Builder
	for i, item := range items {
		if i > 0 {
			builder.WriteString(`<div style="height:12px;line-height:12px;font-size:12px;">&nbsp;</div>`)
		}
		builder.WriteString(renderPoReadyItemCard(item, fallbackURL))
	}
	return builder.String()
}

func renderPoReadyItemCard(item domain.PoReadyItem, fallbackURL string) string {
	safeName := html.EscapeString(item.Name)
	safeURL := html.EscapeString(itemURLOrFallback(item.URL, fallbackURL))
	safeImageURL := html.EscapeString(item.ImageURL)

	imgHTML := `&nbsp;`
	if safeImageURL != "" {
		imgHTML = fmt.Sprintf(
			`<img src="%s" alt="%s" width="76" height="76" style="display:block;width:76px;height:76px;object-fit:contain;border:0;margin:0 auto;">`,
			safeImageURL, safeName,
		)
	}

	// A discounted item shows the live price with the original struck through.
	priceHTML := fmt.Sprintf(`<span style="color:#fc4c02;font-size:18px;font-weight:900;">%s</span>`, formatIDR(item.Price))
	if item.DiscountPrice > 0 && item.DiscountPrice < item.Price {
		priceHTML = fmt.Sprintf(
			`<span style="color:#fc4c02;font-size:18px;font-weight:900;">%s</span>&nbsp;<span style="color:#a2988a;font-size:13px;font-weight:700;text-decoration:line-through;">%s</span>`,
			formatIDR(item.DiscountPrice), formatIDR(item.Price),
		)
	}

	// Pre-Order (struck) → Ready Stock transition badges + name + price.
	inner := fmt.Sprintf(`<table role="presentation" width="100%%" cellspacing="0" cellpadding="0" style="width:100%%;border-collapse:separate;background:#ffffff;border:1px solid #e4dccd;border-radius:12px;">`+
		`<tr>`+
		`<td width="108" valign="middle" align="center" style="width:108px;padding:16px;">`+
		`<table role="presentation" width="76" cellspacing="0" cellpadding="0" align="center" style="width:76px;border-collapse:collapse;background:#f4f0e8;border:1px solid #ece5d8;border-radius:8px;"><tr><td align="center" valign="middle" style="padding:6px;">%s</td></tr></table>`+
		`</td>`+
		`<td valign="middle" align="left" style="padding:16px 18px 16px 0;">`+
		`<span style="display:inline-block;background:#eef0f2;color:#8a9199;font-size:10px;font-weight:800;padding:3px 8px;border-radius:5px;text-decoration:line-through;vertical-align:middle;">Pre-Order</span>`+
		`<span style="color:#fc4c02;font-size:14px;font-weight:900;vertical-align:middle;padding:0 7px;">&rarr;</span>`+
		`<span style="display:inline-block;background:#eafaf1;color:#2e9c5f;font-size:10px;font-weight:900;padding:3px 9px;border-radius:5px;vertical-align:middle;">Ready Stock</span>`+
		`<div style="margin-top:9px;color:#231f1b;font-size:16px;font-weight:800;line-height:1.3;">%s</div>`+
		`<div style="margin-top:6px;line-height:1;">%s</div>`+
		`</td>`+
		`</tr></table>`,
		imgHTML, safeName, priceHTML,
	)

	if safeURL == "" {
		return inner
	}
	return fmt.Sprintf(`<a href="%s" style="display:block;color:inherit;text-decoration:none;">%s</a>`, safeURL, inner)
}
