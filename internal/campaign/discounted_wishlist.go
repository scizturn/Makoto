package campaign

import (
	"fmt"
	"html"
	"math/rand"
	"net/url"
	"strconv"
	"strings"
	texttemplate "text/template"
	"time"

	"github.com/kyou-id/makoto/internal/domain"
)

type DiscountedWishlistCampaign struct {
	TemplateIDs    []string
	Subjects       []string
	Greetings      []string
	WishlistURL    string
	Closing        string
	RandomIntn     func(n int) int
}

const discountedWishlistPromoLimit = 12

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

func (c DiscountedWishlistCampaign) SelectSubject(now time.Time, key string) string {
	if len(c.Subjects) == 0 {
		return ""
	}
	randomIntn := c.RandomIntn
	if randomIntn == nil {
		randomIntn = rand.New(rand.NewSource(templateSeed(now, key+"_subject"))).Intn
	}
	return c.Subjects[randomIntn(len(c.Subjects))]
}

func (c DiscountedWishlistCampaign) RenderSubject(tpl string, user domain.User) (string, error) {
	firstName := user.Name
	if i := strings.Index(user.Name, " "); i > 0 {
		firstName = user.Name[:i]
	}
	t, err := texttemplate.New("subject").Parse(tpl)
	if err != nil {
		return tpl, err
	}
	var buf strings.Builder
	if err := t.Execute(&buf, struct{ Name, FirstName string }{user.Name, firstName}); err != nil {
		return tpl, err
	}
	return buf.String(), nil
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
	featuredHTML := ""
	remainingWishlisted := wishlisted
	remainingFill := fill

	if len(remainingWishlisted) > 0 {
		featuredHTML = renderDiscountedFeaturedCard(remainingWishlisted[0], firstName)
		remainingWishlisted = remainingWishlisted[1:]
	} else if len(remainingFill) > 0 {
		featuredHTML = renderDiscountedFeaturedCard(remainingFill[0], firstName)
		remainingFill = remainingFill[1:]
	}

	displayWishlist, displayFill := capDiscountedWishlistPromoItems(remainingWishlisted, remainingFill, discountedWishlistPromoLimit)

	displayWishlistCount := len(displayWishlist)
	displayFillCount := len(displayFill)
	if featuredHTML != "" {
		if len(wishlisted) > 0 {
			displayWishlistCount++
		} else if len(fill) > 0 {
			displayFillCount++
		}
	}
	// Extract the discount name from the first available item, stripping brackets.
	discountName := ""
	if len(items) > 0 {
		discountName = strings.Trim(items[0].DiscountName, "[]")
	}

	return map[string]any{
		"name":                   user.Name,
		"first_name":             firstName,
		"greeting":               greeting,
		"discount_name":          discountName,
		"wishlist_html":          RenderDiscountedItemsHTML(wishlisted),
		"fill_html":              RenderDiscountedItemsHTML(fill),
		"featured_html":          featuredHTML,
		"promo_html":             RenderDiscountedPromoGridHTML(displayWishlist, displayFill),
		"wishlist_count":         len(wishlisted),
		"fill_count":             len(fill),
		"promo_count":            displayWishlistCount + displayFillCount,
		"display_wishlist_count": displayWishlistCount,
		"display_fill_count":     displayFillCount,
		"wishlist_url":           c.WishlistURL,
		"closing":                c.Closing,
		// Unsubscribe di-handle oleh Kirim.email (List-Unsubscribe / footer provider),
		// jadi Makoto tidak lagi merender link unsubscribe sendiri.
		// Footer CTA khusus discounted wishlist: "Explore kyou yuk!" ke homepage.
		"footer_html": RenderMemberversaryFooterHTMLWithCTA(c.Closing, "https://kyou.id/", "Explore kyou yuk!"),
	}
}

func capDiscountedWishlistPromoItems(wishlisted, fill []domain.DiscountedWishlistItem, limit int) ([]domain.DiscountedWishlistItem, []domain.DiscountedWishlistItem) {
	if limit <= 0 {
		return nil, nil
	}
	displayWishlist := wishlisted
	if len(displayWishlist) > limit {
		displayWishlist = displayWishlist[:limit]
	}
	remaining := limit - len(displayWishlist)
	displayFill := fill
	if remaining < len(displayFill) {
		displayFill = displayFill[:remaining]
	}
	return displayWishlist, displayFill
}

func firstDiscountedWishlistItem(groups ...[]domain.DiscountedWishlistItem) (domain.DiscountedWishlistItem, bool) {
	for _, group := range groups {
		if len(group) > 0 {
			return group[0], true
		}
	}
	return domain.DiscountedWishlistItem{}, false
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
			builder.WriteString(`<td width="212" valign="top" align="center" style="width:212px;padding:0 7px;text-align:center;">`)
			builder.WriteString(renderDiscountedItemCard(items[i]))
			builder.WriteString(`</td>`)
		}
		for i := end; i < start+3; i++ {
			builder.WriteString(`<td width="212" valign="top" align="center" style="width:212px;padding:0 7px;text-align:center;">&nbsp;</td>`)
		}
		builder.WriteString(`</tr>`)
	}

	builder.WriteString(`</table>`)
	return builder.String()
}

func RenderDiscountedPromoGridHTML(wishlisted, fill []domain.DiscountedWishlistItem) string {
	items := append([]domain.DiscountedWishlistItem{}, wishlisted...)
	items = append(items, fill...)
	if len(items) == 0 {
		return `<p style="margin:0;color:#6b7280;">Nantikan promo menarik berikutnya!</p>`
	}

	var builder strings.Builder
	builder.WriteString(`<table role="presentation" width="636" cellspacing="0" cellpadding="0" align="center" style="width:636px;border-collapse:collapse;margin:0 auto;">`)
	for rowStart := 0; rowStart < len(items); rowStart += 3 {
		rowEnd := rowStart + 3
		if rowEnd > len(items) {
			rowEnd = len(items)
		}
		if rowStart > 0 {
			builder.WriteString(`<tr><td colspan="3" style="height:20px;line-height:20px;font-size:20px;">&nbsp;</td></tr>`)
		}
		builder.WriteString(`<tr>`)
		for i := rowStart; i < rowEnd; i++ {
			builder.WriteString(`<td width="212" valign="top" align="center" style="width:212px;padding:5px 7px;text-align:center;">`)
			builder.WriteString(renderDiscountedPromoCard(items[i], i))
			builder.WriteString(`</td>`)
		}
		for i := rowEnd; i < rowStart+3; i++ {
			builder.WriteString(`<td width="212" valign="top" align="center" style="width:212px;padding:0 7px;text-align:center;">&nbsp;</td>`)
		}
		builder.WriteString(`</tr>`)
	}
	builder.WriteString(`</table>`)
	return builder.String()
}

func renderDiscountedFeaturedCard(item domain.DiscountedWishlistItem, firstName string) string {
	mainName, version := discountedWishlistDisplayName(item)
	safeName := html.EscapeString(mainName)
	safeVersion := html.EscapeString(version)
	safeFullName := safeName
	safeSeries := html.EscapeString(displayManufacturerOrFallback(item.SeriesName, "Koleksi"))
	safeURL := html.EscapeString(itemURLOrFallback(item.URL, "https://kyou.id/user/wishlist"))
	safeImageURL := html.EscapeString(item.ImageURL)

	imgHTML := fmt.Sprintf(
		`<img src="%s" alt="%s" width="190" height="190" style="display:block;width:190px;height:190px;object-fit:cover;background:#f3f4f6;border:0;border-radius:12px;">`,
		safeImageURL, safeName,
	)
	if safeImageURL == "" {
		imgHTML = `<div style="width:190px;height:190px;background:#f3f4f6;border-radius:12px;"></div>`
	}

	versionText := ""
	if safeVersion != "" {
		versionText = " " + safeVersion
	}
	seriesText := ""
	if safeSeries != "" {
		seriesText = fmt.Sprintf(`&nbsp;dari %s`, safeSeries)
	}
	safeFirstName := html.EscapeString(strings.TrimSpace(firstName))
	nameText := ""
	if safeFirstName != "" {
		nameText = ", " + safeFirstName
	}

	priceText := formatIDR(effectiveDiscountedPrice(item))
	if priceText == "" {
		priceText = "harga promo"
	}

	// "Dia lagi diskon IDR <potongan>, jadi hanya IDR <harga baru>." The saving is
	// only meaningful when an original price exists and actually differs.
	priceSentence := fmt.Sprintf(`Sekarang hanya&nbsp;<strong style="color:#fc4c02;">%s</strong>.`, priceText)
	if item.DiscountPrice > 0 && item.OriginalPrice > item.DiscountPrice {
		if savingText := formatIDR(item.OriginalPrice - item.DiscountPrice); savingText != "" {
			priceSentence = fmt.Sprintf(`Dia lagi diskon&nbsp;<strong style="color:#2d2d2d;">%s</strong>, jadi hanya&nbsp;<strong style="color:#fc4c02;">%s</strong>.`, savingText, priceText)
		}
	}

	return fmt.Sprintf(
		`<div style="width:660px;margin:0 auto;"><table role="presentation" width="660" cellspacing="0" cellpadding="0" style="width:660px;border-collapse:separate;border-spacing:0;border:2px solid #fc4c02;border-radius:12px;background:#ffffff;overflow:hidden;"><tr><td width="224" valign="top" style="width:224px;padding:17px 0 17px 17px;">%s</td><td width="436" valign="middle" style="width:436px;padding:20px 24px 20px 14px;"><div style="margin-bottom:12px;"><span style="display:inline-block;padding:6px 14px;border-radius:999px;background:#fc4c02;color:#ffffff;font-size:11px;font-weight:900;line-height:1.2;letter-spacing:1px;">Top Picks dari Wishlistmu &#128081;</span></div><h2 style="margin:0 0 8px;color:#2d2d2d;font-size:18px;font-weight:900;line-height:1.2;">Kouka simpenin ini khusus untuk kamu, karena kayanya ini spesial buat kamu!</h2><p style="margin:0 0 16px;color:#565252;font-size:14px;font-weight:600;line-height:1.5;">Kamu suka&nbsp;<strong style="color:#2d2d2d;">%s%s</strong>%s&nbsp;kan%s? %s</p><a href="%s" style="display:inline-block;padding:12px 24px;border-radius:8px;background:#fc4c02;color:#ffffff;font-size:14px;font-weight:900;line-height:1;text-decoration:none;">Aku mau %s!</a></td></tr></table></div>`,
		imgHTML,
		safeFullName, versionText,
		seriesText, nameText, priceSentence,
		safeURL,
		safeFullName,
	)
}

func renderDiscountedPromoCard(item domain.DiscountedWishlistItem, index int) string {
	mainName, version := discountedWishlistDisplayName(item)
	safeName := html.EscapeString(mainName)
	safeVersion := html.EscapeString(version)
	safeSeries := html.EscapeString(displayManufacturerOrFallback(item.SeriesName, "Koleksi"))
	safeURL := html.EscapeString(item.URL)
	safeImageURL := html.EscapeString(item.ImageURL)

	imgHTML := fmt.Sprintf(
		`<img src="%s" alt="%s" width="166" height="166" style="display:block;width:166px;height:166px;object-fit:cover;background:#f3f4f6;border:0;">`,
		safeImageURL, safeName,
	)
	if safeImageURL == "" {
		imgHTML = `<div style="width:166px;height:166px;background:#f3f4f6;"></div>`
	}

	border := "1px solid #e8e1d6"
	label := "BUAT KAMU"
	labelBackground := "#f3f4f6"
	labelColor := "#565252"
	if item.IsWishlisted {
		border = "2px solid #ff5a24"
		label = "&hearts; WISHLIST"
		labelBackground = "#fff3ee"
		labelColor = "#fc4c02"
	}

	versionP := ""
	if safeVersion != "" {
		versionP = fmt.Sprintf(`<p style="margin:2px 0 0;color:#565252;font-size:9px;font-weight:700;line-height:1.2;">%s</p>`, safeVersion)
	}
	originalPriceHTML := ""
	if item.OriginalPrice > 0 && item.DiscountPrice > 0 && item.OriginalPrice != item.DiscountPrice {
		originalPriceHTML = fmt.Sprintf(`<p style="margin:7px 0 1px;color:#9a9a9a;font-size:10px;font-weight:700;line-height:1.2;text-decoration:line-through;">%s</p>`, formatIDR(item.OriginalPrice))
	}
	pctBadge := ""
	if pct := discountPercent(item); pct > 0 {
		pctBadge = fmt.Sprintf(`<span style="display:inline-block;margin-left:4px;padding:2px 5px;border-radius:4px;background:#ff5a24;color:#ffffff;font-size:8px;font-weight:900;line-height:1;">-%d%%</span>`, pct)
	}
	priceHTML := ""
	if price := effectiveDiscountedPrice(item); price > 0 {
		priceMargin := "7px 0 0"
		if originalPriceHTML != "" {
			priceMargin = "0"
		}
		priceHTML = fmt.Sprintf(`%s<p style="margin:%s;color:#2d2d2d;font-size:13px;font-weight:900;line-height:1.2;letter-spacing:0;">%s%s</p>`, originalPriceHTML, priceMargin, formatIDR(price), pctBadge)
	}

	rotations := []string{"-1.4deg", "1.1deg", "-0.7deg", "1.3deg", "-1deg", "0.8deg"}
	rotation := rotations[index%len(rotations)]
	inner := fmt.Sprintf(
		`<div style="position:relative;width:190px;margin:0 auto;padding-top:9px;transform:rotate(%s);"><div style="position:absolute;z-index:2;left:0;width:100%%;top:4px;text-align:center;"><span style="display:inline-block;width:58px;height:15px;background:#ffb38f;background:repeating-linear-gradient(90deg,rgba(255,130,80,.48) 0,rgba(255,130,80,.48) 7px,rgba(255,196,157,.62) 7px,rgba(255,196,157,.62) 11px);opacity:.82;transform:rotate(-1deg);"></span></div><div style="box-sizing:border-box;width:184px;margin:0 auto;padding:7px 7px 10px;border:%s;background:#ffffff;box-shadow:0 5px 10px rgba(67,50,32,.16);"><div style="width:166px;height:166px;background:#f3f4f6;">%s</div><div style="min-height:94px;padding:8px 1px 0;text-align:left;background:#ffffff;"><div style="margin-bottom:6px;"><span style="display:inline-block;padding:3px 7px;border-radius:999px;background:%s;color:%s;font-size:8px;font-weight:900;line-height:1;letter-spacing:.5px;">%s</span></div><p style="margin:0 0 4px;color:#a2a2a2;font-size:8px;font-weight:800;line-height:1.2;letter-spacing:.5px;text-transform:uppercase;white-space:nowrap;overflow:hidden;text-overflow:ellipsis;">%s</p><p style="margin:0;color:#2d2d2d;font-size:12px;font-weight:800;line-height:1.22;letter-spacing:0;">%s</p>%s%s</div></div></div>`,
		rotation, border, imgHTML, labelBackground, labelColor, label, safeSeries, safeName, versionP, priceHTML,
	)

	if safeURL == "" {
		return inner
	}
	return fmt.Sprintf(`<a href="%s" style="display:block;width:198px;margin:auto;color:inherit;text-decoration:none;">%s</a>`, safeURL, inner)
}

func renderDiscountedItemCard(item domain.DiscountedWishlistItem) string {
	mainName, version := discountedWishlistDisplayName(item)
	safeName := html.EscapeString(mainName)
	safeVersion := html.EscapeString(version)
	safeSeries := html.EscapeString(displayManufacturerOrFallback(item.SeriesName, "Koleksi"))
	safeURL := html.EscapeString(item.URL)
	safeImageURL := html.EscapeString(item.ImageURL)

	imgHTML := fmt.Sprintf(
		`<img src="%s" alt="%s" width="188" height="188" style="display:block;width:188px;height:188px;object-fit:cover;background:#f3f4f6;border:0;border-radius:8px 8px 0 0;">`,
		safeImageURL, safeName,
	)
	if safeImageURL == "" {
		imgHTML = `<div style="width:188px;height:188px;background:#f3f4f6;border-radius:8px 8px 0 0;"></div>`
	}

	versionP := ""
	if safeVersion != "" {
		versionP = fmt.Sprintf(
			`<p style="margin:0;color:#2d2d2d;font-size:12px;font-weight:800;line-height:1.25;">%s</p>`,
			safeVersion,
		)
	}

	originalPriceHTML := ""
	if item.OriginalPrice > 0 && item.OriginalPrice != item.DiscountPrice {
		originalPriceHTML = fmt.Sprintf(
			`<p style="margin:8px 0 2px;color:#8b8b8b;font-size:11px;font-weight:700;line-height:1.2;text-decoration:line-through;">%s</p>`,
			formatIDR(item.OriginalPrice),
		)
	}

	pctBadge := ""
	if item.OriginalPrice > 0 && item.DiscountPrice > 0 && item.OriginalPrice > item.DiscountPrice {
		pct := (item.OriginalPrice - item.DiscountPrice) * 100 / item.OriginalPrice
		pctBadge = fmt.Sprintf(
			`<span style="display:inline-block;margin-left:6px;padding:4px 7px;background:#fc4c02;border-radius:5px;color:#ffffff;font-size:10px;font-weight:900;line-height:1;">-%d%%</span>`,
			pct,
		)
	}

	discountPriceHTML := ""
	if item.DiscountPrice > 0 {
		discountPriceHTML = fmt.Sprintf(
			`<p style="margin:0;color:#fc4c02;font-size:18px;font-weight:900;line-height:1.2;letter-spacing:0;">%s%s</p>`,
			formatIDR(item.DiscountPrice), pctBadge,
		)
	}

	inner := fmt.Sprintf(
		`<div style="overflow:hidden;width:188px;border-radius:8px;background:#fff4ec;box-shadow:0 8px 18px rgba(45,45,45,0.16);">%s<div style="min-height:106px;padding:12px 14px 14px;text-align:left;background:linear-gradient(180deg,#fff7f1 0%%,#ffe9dc 100%%);"><p style="margin:0 0 5px;color:#8b8b8b;font-size:10px;font-weight:900;line-height:1.2;letter-spacing:0.8px;text-transform:uppercase;white-space:nowrap;overflow:hidden;text-overflow:ellipsis;">%s</p><p style="margin:0;color:#2d2d2d;font-size:13px;font-weight:900;line-height:1.25;">%s</p>%s%s%s</div></div>`,
		imgHTML,
		safeSeries, safeName, versionP,
		originalPriceHTML, discountPriceHTML,
	)

	if safeURL == "" {
		return inner
	}
	return fmt.Sprintf(
		`<a href="%s" style="display:block;width:188px;margin:auto;color:inherit;text-decoration:none;">%s</a>`,
		safeURL, inner,
	)
}

func discountedWishlistDisplayName(item domain.DiscountedWishlistItem) (string, string) {
	if characterName := strings.TrimSpace(item.CharacterName); characterName != "" {
		return characterName, ""
	}
	return abbreviateCardTitle(item.Name, item.SeriesName)
}

func discountPercent(item domain.DiscountedWishlistItem) int {
	if item.OriginalPrice <= 0 || item.DiscountPrice <= 0 || item.OriginalPrice <= item.DiscountPrice {
		return 0
	}
	return (item.OriginalPrice - item.DiscountPrice) * 100 / item.OriginalPrice
}

func effectiveDiscountedPrice(item domain.DiscountedWishlistItem) int {
	if item.DiscountPrice > 0 {
		return item.DiscountPrice
	}
	return item.OriginalPrice
}

func itemURLOrFallback(rawURL, fallback string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return fallback
	}
	if _, err := url.ParseRequestURI(rawURL); err != nil {
		return fallback
	}
	return rawURL
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
	return "IDR " + strings.Join(parts, ".")
}
