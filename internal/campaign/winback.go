package campaign

import (
	"fmt"
	"hash/fnv"
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
		// "sejak <tahun>" in the scrapbook cover — the year the user joined Kyou.
		// Falls back to Kyou's founding year when the account date is unknown.
		"join_date": winbackMemberSinceYear(user.CreatedAt),
		// "Album Kenangan" scrapbook from real data. Reuse the existing
		// historical_html/wishlist_html merge keys so no new viewData field is
		// needed — winback templates render their own scrapbook style. The past
		// collection is a list (thumbnail + name + purchase date); the ready
		// wishlist stays a polaroid grid.
		"historical_html": RenderWinbackPastListHTML(historicalItems),
		"wishlist_html":   RenderWinbackReadyStampsHTML(wishlist),
		"action_url":      actionURLWithClaim(c.ActionURL, voucherCode),
		"closing":         c.Closing,
		// Shared Kyou footer (same as discounted wishlist).
		"footer_html": RenderMemberversaryFooterHTMLWithCTA(c.Closing, "https://kyou.id/", "Explore kyou yuk!"),
	}
}

const (
	// The ready wishlist/fill grid is a 2-column × 3-row sheet of postage stamps.
	winbackReadyCols  = 2
	winbackReadyLimit = 6
)

// winbackStampMat is the color of the "page" behind each stamp — it must match
// the lined-paper background the winback templates place the grid on so the
// perforation notches (radial-gradient holes) blend into the page.
const winbackStampMat = "#f4ecd7"

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

	// The photo is styled like a printed polaroid taped into a scrapbook: a white
	// frame with a soft drop shadow and a handwritten date caption underneath —
	// the "ini ada memori-nya banget" feel. To force a hard 1:1 square in every
	// renderer, the photo lives inside a fixed 170x170 box with overflow:hidden
	// (box dimensions are honored far more widely than <img> height). object-fit:
	// cover crops the photo to fill that square in clients that support it (iOS
	// Mail, Gmail, Apple Mail). The Caveat handwriting font is loaded by the
	// winback templates this HTML injects into.
	photoInner := ""
	if safeImage != "" {
		photoInner = fmt.Sprintf(`<img src="%s" alt="" width="170" height="170" style="display:block;width:170px;height:170px;object-fit:cover;border:0;">`, safeImage)
	}
	photo := fmt.Sprintf(`<div style="width:170px;height:170px;overflow:hidden;background:#efe7d3;font-size:0;line-height:0;margin:0 auto;">%s</div>`, photoInner)
	handDate := ""
	if dateText := winbackOrderDateText(item.OrderDate); dateText != "" {
		// white-space:nowrap keeps the caption on a single line; the polaroid frame
		// below grows to fit it, so the (randomized, longer) phrases never wrap.
		handDate = fmt.Sprintf(`<div style="font-family:'Caveat','Segoe Print','Bradley Hand',cursive;font-weight:700;font-size:13px;color:#9a6f45;text-align:center;line-height:1.15;white-space:nowrap;padding:8px 4px 1px;">&hearts; %s</div>`, html.EscapeString(winbackPastCaption(item.Name, dateText)))
	}
	// The polaroid grows to the wider of the 170px photo and the handwritten
	// caption (display:inline-block); the photo is centered via margin:0 auto and
	// the caption via text-align:center, so both stay aligned when the frame grows.
	// The parent <td> is text-align:center so the whole card centers in its column.
	polaroid := fmt.Sprintf(`<div style="display:inline-block;background:#ffffff;padding:9px 9px 6px;border:1px solid #e6dcc6;box-shadow:2px 3px 8px rgba(74,59,42,0.22);">%s%s</div>`, photo, handDate)

	// Right column: item name on top, series name underneath (kalau ada).
	nameBlock := fmt.Sprintf(`<div style="font-family:'Nunito',Arial,Helvetica,sans-serif;font-weight:800;font-size:18px;color:#4a3b2a;line-height:1.35;">%s</div>`, safeName)
	if series := strings.TrimSpace(item.SeriesName); series != "" {
		nameBlock += fmt.Sprintf(`<div style="font-family:'Nunito',Arial,Helvetica,sans-serif;font-weight:600;font-size:13.5px;color:#9a866a;margin-top:5px;">%s</div>`, html.EscapeString(series))
	}

	row := fmt.Sprintf(
		`<table role="presentation" width="100%%" cellspacing="0" cellpadding="0" style="border-collapse:separate;border-radius:16px;background:#fffdf6;border:1px solid #ece3d0;box-shadow:1px 2px 6px rgba(74,59,42,0.12);overflow:hidden;"><tr><td width="220" valign="middle" style="width:220px;padding:20px 8px 20px 14px;text-align:center;">%s</td><td valign="middle" style="padding:16px 22px;">%s</td></tr></table>`,
		polaroid, nameBlock,
	)
	// Each row links to the item so the reader can jump straight to its page.
	if strings.TrimSpace(item.URL) == "" {
		return row
	}
	return fmt.Sprintf(`<a href="%s" style="text-decoration:none;color:inherit;display:block;">%s</a>`, html.EscapeString(item.URL), row)
}

// winbackListCaption cleans an item name for the wider list layout: it strips
// leading bracket tags (e.g. "[PSA 10] [with Bonus] Foo" → "Foo") and truncates
// what remains. The stamp grid uses the tighter winbackStampCaption.
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

// winbackFoundingYear is Kyou's founding year, used as the "sejak <year>"
// fallback when a user's account creation date is unknown (zero time).
const winbackFoundingYear = "2014"

// winbackMemberSinceYear returns the four-digit year the user joined Kyou for
// the scrapbook cover's "sejak <year>" line. A zero createdAt (missing on jobs
// enqueued before the created_at field existed) falls back to the founding year.
func winbackMemberSinceYear(createdAt time.Time) string {
	if createdAt.IsZero() {
		return winbackFoundingYear
	}
	return fmt.Sprintf("%d", createdAt.Year())
}

// winbackOrderDateText formats a purchase date in Indonesian day-month-year,
// e.g. "1 Mei 2026". Returns "" for a zero date. The surrounding caption phrase
// is added by winbackPastCaption.
func winbackOrderDateText(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return fmt.Sprintf("%d %s %d", t.Day(), idMonths[t.Month()], t.Year())
}

// winbackPastCaptionPhrases are the handwritten caption variants shown under
// each past-purchase polaroid; %s is the formatted purchase date. One is picked
// per item (hashed on the item name) so a retry of the same job renders
// identically.
var winbackPastCaptionPhrases = []string{
	"Menemani sejak %s",
}

// winbackPastCaption builds one item's handwritten caption, choosing a phrase
// deterministically from the item name so retries stay identical.
func winbackPastCaption(name, dateText string) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(name))
	phrase := winbackPastCaptionPhrases[h.Sum32()%uint32(len(winbackPastCaptionPhrases))]
	return fmt.Sprintf(phrase, dateText)
}

// RenderWinbackReadyStampsHTML renders the ready wishlist/fill items as a
// 2-column × 3-row sheet of postage stamps ("wishlist kamu, ready stock").
func RenderWinbackReadyStampsHTML(items []domain.WishlistItem) string {
	// Drop "wakeari" (imperfect-grade) items before capping, so they never take a
	// stamp slot even if they slipped in via the user's own wishlist or a job
	// enqueued before winback_fill_items.sql filtered them out. Adult items are
	// already excluded upstream in SQL (isAdult=0).
	kept := items[:0:0]
	for _, item := range items {
		if isWakeariName(item.Name) {
			continue
		}
		kept = append(kept, item)
	}
	items = kept
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
			b.WriteString(fmt.Sprintf(`<tr><td colspan="%d" style="height:18px;line-height:18px;font-size:18px;">&nbsp;</td></tr>`, winbackReadyCols))
		}
		b.WriteString(`<tr>`)
		end := row + winbackReadyCols
		if end > len(items) {
			end = len(items)
		}
		for i := row; i < end; i++ {
			b.WriteString(`<td width="50%" valign="top" align="center" style="padding:0 8px;">` + winbackStampCard(items[i]) + `</td>`)
		}
		for i := end; i < row+winbackReadyCols; i++ {
			b.WriteString(`<td width="50%">&nbsp;</td>`)
		}
		b.WriteString(`</tr>`)
	}
	b.WriteString(`</table>`)
	return b.String()
}

// winbackStampCard renders one wishlist item as a postage stamp: a cream sheet
// with perforated edges (radial-gradient holes in the page color, revealed only
// in the padding ring around a solid inner panel), the item photo, and an
// issuer/denomination banner along the bottom — like a real perangko. It avoids
// position:absolute and transform (Outlook strips them); where radial-gradient
// is unsupported it degrades to a clean cream card, so no client breaks.
func winbackStampCard(item domain.WishlistItem) string {
	safeImage := html.EscapeString(item.ImageURL)

	// Square 1:1 crop to match how these items appear as most-popular thumbnails
	// on /search. aspect-ratio + object-fit give an exact square in modern clients
	// (Apple Mail, iOS, Gmail); Outlook ignores them and shows the image at its
	// natural ratio — still visible, just not cropped.
	imgHTML := fmt.Sprintf(`<img src="%s" alt="" width="300" style="display:block;width:100%%;height:auto;aspect-ratio:1/1;object-fit:cover;border:0;background:#efe7d3;">`, safeImage)
	if safeImage == "" {
		imgHTML = `<div style="width:100%;aspect-ratio:1/1;background:#efe7d3;"></div>`
	}

	// The user's own wishlist items get an orange issuer banner; popular-ready
	// fill items get the muted brown banner so they stand apart.
	bannerBG := "#4a3b2a"
	issuer := "KYOU.ID"
	if item.IsWishlisted {
		bannerBG = "#fc4c02"
		issuer = "&hearts; WISHLIST-KU"
	}

	banner := fmt.Sprintf(
		`<tr><td style="background:%s;padding:7px 11px;"><span style="font-family:'Nunito',Arial,Helvetica,sans-serif;font-weight:800;font-size:11px;letter-spacing:2px;color:#fbe6d0;">%s</span></td></tr>`,
		bannerBG, issuer,
	)

	// Outer table paints mat-colored perforation dots over cream; the inner panel
	// (cream) covers the center, so the holes only show in the 9px padding ring.
	perf := fmt.Sprintf("background-color:#fffdf6;background-image:radial-gradient(circle,%s 3px,rgba(244,236,215,0) 3.5px);background-size:13px 13px;background-position:center;", winbackStampMat)
	card := fmt.Sprintf(
		`<table role="presentation" width="100%%" cellspacing="0" cellpadding="0" style="max-width:300px;margin:0 auto;border-collapse:collapse;%sbox-shadow:2px 4px 9px rgba(74,59,42,0.22);"><tr><td style="padding:9px;"><table role="presentation" width="100%%" cellspacing="0" cellpadding="0" style="border-collapse:collapse;background:#fffdf6;border:1px solid #e3d7bb;"><tr><td style="padding:0;font-size:0;line-height:0;">%s</td></tr>%s</table></td></tr></table>`,
		perf, imgHTML, banner,
	)

	caption := fmt.Sprintf(
		`<div style="font-family:'Nunito',Arial,Helvetica,sans-serif;font-weight:700;font-size:13px;color:#4a3b2a;text-align:center;padding:9px 4px 0;line-height:1.3;max-width:300px;margin:0 auto;">%s</div>`,
		html.EscapeString(winbackStampCaption(item.Name)),
	)

	stamp := card + caption
	if strings.TrimSpace(item.URL) == "" {
		return stamp
	}
	return fmt.Sprintf(`<a href="%s" style="text-decoration:none;color:inherit;display:block;">%s</a>`, html.EscapeString(item.URL), stamp)
}

// isWakeariName reports whether an item name carries the "wakeari" (訳あり /
// imperfect-grade) tag. There is no dedicated flag on the item, so the name is
// the only signal.
func isWakeariName(name string) bool {
	return strings.Contains(strings.ToLower(name), "wakeari")
}

// winbackStampCaption cleans an item name for the stamp grid: it strips leading
// bracket tags (e.g. "[PSA 10] Foo" → "Foo") but shows the full remaining name —
// no truncation — so the fill items read in full under each stamp.
func winbackStampCaption(name string) string {
	return stripBracketPrefixes(name)
}
