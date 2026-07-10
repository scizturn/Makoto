package campaign

import (
	"fmt"
	"hash/fnv"
	"html"
	"math/rand"
	"net/url"
	"strings"
	texttemplate "text/template"
	"time"

	"github.com/kyou-id/makoto/internal/domain"
)

type BirthdayCampaign struct {
	TemplateIDs []string
	Subjects    []string
	Closing     string
	ActionURL   string
	RandomIntn  func(n int) int
}

func (c BirthdayCampaign) SelectTemplate(now time.Time, key string) string {
	if len(c.TemplateIDs) == 0 {
		return ""
	}
	randomIntn := c.RandomIntn
	if randomIntn == nil {
		randomIntn = rand.New(rand.NewSource(templateSeed(now, key))).Intn
	}
	return c.TemplateIDs[randomIntn(len(c.TemplateIDs))]
}

// SelectSubject draws from the same seed as SelectTemplate, so subject i always
// ships with design i.
func (c BirthdayCampaign) SelectSubject(now time.Time, key string) string {
	if len(c.Subjects) == 0 {
		return ""
	}
	randomIntn := c.RandomIntn
	if randomIntn == nil {
		randomIntn = rand.New(rand.NewSource(templateSeed(now, key))).Intn
	}
	return c.Subjects[randomIntn(len(c.Subjects))]
}

func (c BirthdayCampaign) RenderSubject(tpl string, user domain.User) (string, error) {
	firstName := user.Name
	if i := strings.Index(user.Name, " "); i > 0 {
		firstName = user.Name[:i]
	}
	t, err := texttemplate.New("subject").Parse(tpl)
	if err != nil {
		return "", err
	}
	var buf strings.Builder
	if err := t.Execute(&buf, subjectData{Name: user.Name, FirstName: firstName}); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func templateSeed(now time.Time, key string) int64 {
	hash := fnv.New64a()
	_, _ = hash.Write([]byte(now.Format(time.RFC3339Nano)))
	_, _ = hash.Write([]byte("|"))
	_, _ = hash.Write([]byte(strings.TrimSpace(key)))
	return int64(hash.Sum64())
}

func (c BirthdayCampaign) BuildMergeData(user domain.User, voucherCode string, wishlist []domain.WishlistItem, fyp []domain.FYPItem, popular []domain.FYPItem) map[string]any {
	fypItems := topUpFYPItems(fyp, popular, 3)

	return map[string]any{
		"name":           user.Name,
		"voucher_code":   voucherCode,
		"wishlist_items": wishlist,
		"fyp_items":      fypItems,
		"wishlist_html":  RenderWishlistHTML(wishlist),
		"fyp_html":       RenderFYPHTML(fypItems),
		"action_url":     actionURLWithClaim(c.ActionURL, voucherCode),
		"closing":        c.Closing,
	}
}

func actionURLWithClaim(baseURL string, voucherCode string) string {
	if strings.TrimSpace(baseURL) == "" || strings.TrimSpace(voucherCode) == "" {
		return baseURL
	}
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return baseURL
	}
	query := parsed.Query()
	query.Set("claim", voucherCode)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func topUpFYPItems(fyp []domain.FYPItem, popular []domain.FYPItem, limit int) []domain.FYPItem {
	if limit <= 0 {
		return nil
	}
	items := make([]domain.FYPItem, 0, limit)
	seen := make(map[string]bool)
	for _, item := range fyp {
		if len(items) >= limit {
			break
		}
		if item.ID != "" {
			if seen[item.ID] {
				continue
			}
			seen[item.ID] = true
		}
		items = append(items, item)
	}
	for _, item := range popular {
		if len(items) >= limit {
			break
		}
		if item.ID != "" {
			if seen[item.ID] {
				continue
			}
			seen[item.ID] = true
		}
		items = append(items, item)
	}
	return items
}

func RenderWishlistHTML(items []domain.WishlistItem) string {
	if len(items) == 0 {
		return `<p style="margin:0;color:#6b7280;">Your wishlist is waiting for your next pick.</p>`
	}

	return renderWishlistFeature(items[0])
}

func RenderWishlistGridHTML(items []domain.WishlistItem) string {
	if len(items) == 0 {
		return `<p style="margin:0;color:#6b7280;">Your wishlist is waiting for your next pick.</p>`
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

func FYPToWishlistItem(p domain.FYPItem) domain.WishlistItem {
	return domain.WishlistItem{
		ID:           p.ID,
		Name:         p.Name,
		URL:          "https://kyou.id/items/" + p.ID + "/",
		ImageURL:     p.ImageURL,
		Price:        p.Price,
		Status:       p.Status,
		Manufacturer: p.Manufacturer,
		SeriesName:   p.SeriesName,
		PODeadline:   p.PODeadline,
		POReleaseAt:  p.POReleaseAt,
	}
}

func RenderFYPHTML(items []domain.FYPItem) string {
	if len(items) == 0 {
		return `<p style="margin:0;color:#6b7280;">Recommended picks are being refreshed for you.</p>`
	}

	var builder strings.Builder
	builder.WriteString(`<table role="presentation" width="690" cellspacing="0" cellpadding="0" align="center" style="width:690px;border-collapse:collapse;margin:0 auto;">`)
	builder.WriteString(`<tr>`)
	for index, item := range items {
		if index >= 3 {
			break
		}
		kind := item.Kind
		if strings.TrimSpace(kind) == "" {
			kind = "Recommended"
		}
		builder.WriteString(`<td width="230" valign="top" align="center" style="width:230px;padding:0;text-align:center;">`)
		builder.WriteString(renderProductCard(item.Name, item.SeriesName, kind, itemURL(item.ID), item.ImageURL))
		builder.WriteString(`</td>`)
	}
	builder.WriteString(`</tr></table>`)
	return builder.String()
}

func renderWishlistFeature(item domain.WishlistItem) string {
	displayName, version := splitItemDisplayName(item.Name, item.SeriesName)
	safeName := html.EscapeString(displayName)
	safeVersion := html.EscapeString(version)
	safeSeries := html.EscapeString(displayWishlistSeries(item.SeriesName, item.Manufacturer))
	safeURL := html.EscapeString(item.URL)
	safeImageURL := html.EscapeString(item.ImageURL)
	statusHTML := statusBadgeHTML(itemDisplayStatus(item.Status, item.PODeadline))
	imageHTML := `<div style="width:170px;height:170px;background:#f3e7d8;border-radius:6px;"></div>`
	if safeImageURL != "" {
		imageHTML = fmt.Sprintf(`<img src="%s" alt="%s" width="170" height="170" style="display:block;width:170px;height:170px;object-fit:cover;background:#f3e7d8;border:0;border-radius:6px;">`, safeImageURL, safeName)
	}

	nameHTML := safeName
	versionHTML := ""
	if safeVersion != "" {
		versionHTML = fmt.Sprintf(`<p style="margin:5px 0 0;color:#ffffff;font-size:19px;font-weight:800;line-height:1.22;">%s</p>`, safeVersion)
	}

	cardHTML := fmt.Sprintf(`<div style="margin:0 auto;width:600px;padding:26px 30px;border:2px solid #7a4a24;border-radius:18px;background:#321b10;color:#ffffff;"><table role="presentation" width="600" cellspacing="0" cellpadding="0" style="width:600px;border-collapse:collapse;"><tr><td width="170" valign="middle" style="width:170px;padding:0 42px 0 0;"><div style="padding:6px;border:5px solid #fff18f;border-radius:12px;background:#fff8d1;">%s</div></td><td width="404" valign="middle" style="width:404px;padding:0 0 0 0;"><p style="margin:0 0 4px;color:#ffd68a;font-size:14px;font-weight:900;letter-spacing:3px;text-transform:uppercase;">%s</p><h3 style="margin:0 0 16px;color:#ffffff;font-size:19px;font-weight:800;line-height:1.22;">%s%s</h3><p style="margin:0 0 16px;color:#f7dfce;font-size:16px;font-weight:650;line-height:1.52;">Salah satu wishlist idaman kamu lagi nunggu buat jadi bagian dari koleksimu. Yuk, wujudkan bareng Kyou sekarang!</p><span style="display:inline-block;margin:0 12px 8px 0;color:#ffbe72;font-size:15px;font-weight:900;letter-spacing:2px;">Wishlist Pick</span>%s</td></tr></table></div>`, imageHTML, safeSeries, nameHTML, versionHTML, statusHTML)
	if safeURL == "" {
		return cardHTML
	}
	return fmt.Sprintf(`<a href="%s" style="display:block;color:inherit;text-decoration:none;">%s</a>`, safeURL, cardHTML)
}

func renderProductCard(name string, seriesName string, status string, url string, imageURL string) string {
	mainName, version := abbreviateCardTitle(name, seriesName)
	safeName := html.EscapeString(mainName)
	safeVersion := html.EscapeString(version)
	safeSeries := html.EscapeString(displayManufacturerOrFallback(seriesName, "Wishlist"))
	safeURL := html.EscapeString(url)
	safeImageURL := html.EscapeString(imageURL)
	_ = status

	imgHTML := fmt.Sprintf(
		`<img src="%s" alt="%s" width="180" height="180" style="display:block;width:180px;height:180px;object-fit:cover;background:#f3f4f6;border:0;">`,
		safeImageURL, safeName,
	)
	if safeImageURL == "" {
		imgHTML = `<div style="width:180px;height:180px;background:#f3f4f6;"></div>`
	}

	versionP := ""
	if safeVersion != "" {
		versionP = fmt.Sprintf(
			`<p style="margin:0;color:#374151;font-size:10px;font-weight:600;line-height:1.3;">%s</p>`,
			safeVersion,
		)
	}

	inner := fmt.Sprintf(
		`<div style="overflow:hidden;width:180px;height:287px;border:1px solid #9ca3af;border-radius:8px;background:url('https://kyoucdn.id/static/assets/item%%20bg1.jpg') center/cover no-repeat;">%s<div style="padding:8px 10px 10px;text-align:left;"><div style="height:52px;overflow:hidden;"><p style="margin:0 0 2px;color:#2f2b28;font-size:10px;font-weight:400;line-height:1.2;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;">%s</p><p style="margin:0 0 2px;color:#0f172a;font-size:12px;font-weight:900;line-height:1.3;white-space:normal;word-break:break-word;">%s</p>%s</div><div style="margin:10px 0 0;text-align:center;"><img src="https://images2.imgbox.com/ef/f8/mAoUYtqE_o.png" alt="Cek item" width="142" style="display:inline-block;width:142px;height:auto;border:0;"></div></div></div>`,
		imgHTML, safeSeries, safeName, versionP,
	)

	if safeURL == "" {
		return inner
	}
	return fmt.Sprintf(
		`<a href="%s" style="display:block;width:180px;margin:auto;color:inherit;text-decoration:none;">%s</a>`,
		safeURL, inner,
	)
}

func abbreviateCardTitle(name, seriesName string) (string, string) {
	// strip [TAG] prefixes, then size "(17cm)" BEFORE series suffix detection
	name = strings.TrimSpace(name)
	for strings.HasPrefix(name, "[") {
		end := strings.Index(name, "]")
		if end < 0 {
			break
		}
		name = strings.TrimSpace(name[end+1:])
	}
	if i := strings.LastIndex(name, " ("); i >= 0 {
		if suf := name[i:]; strings.HasSuffix(suf, ")") && strings.Contains(suf, "cm") {
			name = strings.TrimSpace(name[:i])
		}
	}
	if marker := strings.Index(strings.ToLower(name), " q series "); marker > 0 {
		return strings.TrimSpace(name[:marker]), ""
	}
	name = stripItemNoise(name, seriesName)

	var version string
	if split := strings.LastIndex(name, " - "); split >= 0 && strings.Contains(strings.ToLower(name[split+3:]), "ver") {
		version = cleanVersion(name[split+3:])
		name = strings.TrimSpace(name[:split])
	} else {
		name, version = splitItemVersion(name)
		version = cleanVersion(version)
	}
	// strip any leftover " - Franchise" suffix that has no "ver"
	if split := strings.LastIndex(name, " - "); split >= 0 {
		after := strings.TrimSpace(name[split+3:])
		if !strings.Contains(strings.ToLower(after), "ver") {
			name = strings.TrimSpace(name[:split])
		}
	}
	// handle "X Series" naming (only keep if it contains "Ver.")
	name, sv := splitKnownSeriesVersion(name)
	if version == "" {
		version = cleanVersion(sv)
	}

	prefix := ""
	remainder := name
	for _, p := range []string{
		"Nendoroid Doll", "Nendoroid",
		"Figma",
		"Kuripan Plushie",
		"Xross Link Figure",
		"Espresto Figure",
		"YURUCANA Plastic Model Kit",
		"Plastic Model Kit",
	} {
		if strings.HasPrefix(strings.ToLower(remainder), strings.ToLower(p)) {
			prefix = p
			remainder = strings.TrimSpace(remainder[len(p):])
			break
		}
	}

	if prefix == "" {
		// strip product type prefixes sequentially (PVC Figure + Gift+ can appear together)
		for _, strip := range []string{"PVC Figure", "Gift+"} {
			if strings.HasPrefix(strings.ToLower(remainder), strings.ToLower(strip)) {
				remainder = strings.TrimSpace(remainder[len(strip):])
			}
		}
		// detect leading scale: "1/7", "1/8", "non Scale"
		parts := strings.Fields(remainder)
		if len(parts) > 1 && strings.Contains(parts[0], "/") {
			prefix = parts[0]
			remainder = strings.Join(parts[1:], " ")
		} else if len(parts) > 2 && strings.EqualFold(parts[0], "non") && strings.EqualFold(parts[1], "scale") {
			prefix = parts[0] + " " + parts[1]
			remainder = strings.Join(parts[2:], " ")
		}
	} else {
		// for named types (Nendoroid etc.), also strip a following scale token
		parts := strings.Fields(remainder)
		if len(parts) > 1 && strings.Contains(parts[0], "/") {
			remainder = strings.Join(parts[1:], " ")
		}
	}

	// take last meaningful word (skip trailing pure numbers like "1", "2")
	words := strings.Fields(remainder)
	shortChar := remainder
	for i := len(words) - 1; i >= 0; i-- {
		w := words[i]
		isNum := len(w) > 0
		for _, r := range w {
			if r < '0' || r > '9' {
				isNum = false
				break
			}
		}
		if !isNum || i == 0 {
			shortChar = w
			break
		}
	}

	// if last word is a generic product category (not a character name) and
	// no product prefix was detected, the character name is at the front
	productTerminals := map[string]bool{
		"card": true, "cards": true,
		"box": true, "set": true, "pack": true,
		"badge": true, "plate": true, "stand": true,
		"tapestry": true, "poster": true, "seal": true,
	}
	if prefix == "" && len(words) > 1 && productTerminals[strings.ToLower(shortChar)] {
		shortChar = words[0]
	}

	if prefix != "" {
		return prefix + " " + shortChar, version
	}
	return shortChar, version
}

// cleanVersion returns "" if v has no "Ver", otherwise returns just the
// "...Ver" part — stripping any trailing period and any series name that
// appears after "Ver. SomethingElse".
func cleanVersion(v string) string {
	v = strings.TrimSpace(v)
	lower := strings.ToLower(v)
	vi := strings.Index(lower, "ver")
	if vi < 0 {
		return ""
	}
	// take only up to (and including) "Ver", drop anything after "Ver. X"
	after := v[vi+3:]
	if _, rest, found := strings.Cut(after, "."); found && strings.TrimSpace(rest) != "" {
		v = strings.TrimSpace(v[:vi+3])
	}
	return strings.TrimRight(strings.TrimSpace(v), ".")
}

func displayManufacturer(manufacturer string) string {
	manufacturer = strings.TrimSpace(manufacturer)
	if manufacturer == "" {
		return "Kyou Pick"
	}
	return manufacturer
}

func displayWishlistSeries(seriesName string, manufacturer string) string {
	seriesName = strings.TrimSpace(seriesName)
	if seriesName != "" {
		return strings.ToUpper(seriesName)
	}
	return strings.ToUpper(displayManufacturer(manufacturer))
}

func displayManufacturerOrFallback(seriesName string, fallback string) string {
	seriesName = strings.TrimSpace(seriesName)
	if seriesName != "" {
		return seriesName
	}
	fallback = strings.TrimSpace(fallback)
	if fallback == "" {
		return "Kyou Pick"
	}
	return fallback
}

func splitItemVersion(name string) (string, string) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", ""
	}
	versionMarker := " Ver"
	markerIndex := strings.LastIndex(name, versionMarker)
	if markerIndex < 0 {
		return name, ""
	}
	start := markerIndex
	wordsBeforeMarker := 0
	for start > 0 {
		for start > 0 && name[start-1] == ' ' {
			start--
		}
		wordEnd := start
		for start > 0 && name[start-1] != ' ' {
			start--
		}
		if wordEnd > start {
			wordsBeforeMarker++
		}
		if wordsBeforeMarker >= 2 {
			break
		}
	}
	if start <= 0 {
		return name, ""
	}
	return strings.TrimSpace(name[:start]), strings.TrimSpace(name[start:])
}

func splitItemDisplayName(name string, seriesName string) (string, string) {
	name = stripItemNoise(name, seriesName)
	var displayName, version string
	if split := strings.LastIndex(name, " - "); split >= 0 && strings.Contains(strings.ToLower(name[split+3:]), "ver") {
		displayName = strings.TrimSpace(name[:split])
		version = strings.TrimSpace(name[split+3:])
	} else {
		displayName, version = splitItemVersion(name)
	}
	displayName = stripProductPrefix(displayName)
	displayName, suffixVersion := splitKnownSeriesVersion(displayName)
	if version == "" {
		version = suffixVersion
	}
	return displayName, version
}

func stripItemNoise(name string, seriesName string) string {
	name = strings.TrimSpace(name)
	for strings.HasPrefix(name, "[") {
		end := strings.Index(name, "]")
		if end < 0 {
			break
		}
		name = strings.TrimSpace(name[end+1:])
	}
	seriesName = strings.TrimSpace(seriesName)
	if seriesName != "" {
		for _, suffix := range []string{" - " + seriesName, " " + seriesName} {
			if strings.HasSuffix(strings.ToLower(name), strings.ToLower(suffix)) {
				name = strings.TrimSpace(name[:len(name)-len(suffix)])
				break
			}
		}
	}
	for _, suffix := range []string{" Can Badge Blind Box"} {
		if strings.HasSuffix(strings.ToLower(name), strings.ToLower(suffix)) {
			name = strings.TrimSpace(name[:len(name)-len(suffix)])
		}
	}
	name = stripCategorySuffix(name, []string{
		" Series Acrylic Keychain",
		" Acrylic Keychain",
		" Keychain",
	})
	return strings.TrimSpace(name)
}

func stripCategorySuffix(name string, categories []string) string {
	lowerName := strings.ToLower(name)
	for _, category := range categories {
		index := strings.Index(lowerName, strings.ToLower(category))
		if index >= 0 {
			return strings.TrimSpace(name[:index])
		}
	}
	return name
}

func stripProductPrefix(name string) string {
	prefixes := []string{"PVC Figure", "Gift+"}
	for _, prefix := range prefixes {
		name = strings.TrimSpace(name)
		if strings.HasPrefix(strings.ToLower(name), strings.ToLower(prefix)) {
			name = strings.TrimSpace(name[len(prefix):])
		}
	}
	parts := strings.Fields(name)
	for len(parts) > 0 && strings.Contains(parts[0], "/") {
		parts = parts[1:]
	}
	return strings.Join(parts, " ")
}

func splitKnownSeriesVersion(name string) (string, string) {
	parts := strings.Fields(name)
	if len(parts) < 4 || parts[len(parts)-1] != "Series" {
		return name, ""
	}
	version := strings.Join(parts[len(parts)-3:], " ")
	displayName := strings.Join(parts[:len(parts)-3], " ")
	return displayName, version
}

func itemDisplayStatus(status string, poDeadline *time.Time) string {
	normalized := strings.TrimSpace(status)
	if normalized == "" {
		return "READY"
	}
	if strings.EqualFold(normalized, "PO") && poDeadline == nil {
		return "LPO"
	}
	return strings.ToUpper(normalized)
}

func statusBadgeHTML(status string) string {
	label := "Ready Stock"
	color := "#40b774"
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "PO":
		label = "Pre-Order"
		color = "#657996"
	case "LPO":
		label = "Late Pre-Order"
		color = "#d3647a"
	}
	return fmt.Sprintf(`<span style="display:inline-block;padding:7px 12px;border-radius:7px;background:%s;color:#ffffff;font-size:16px;font-weight:900;line-height:1;">%s</span>`, color, label)
}

func itemURL(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return ""
	}
	return "https://kyou.id/items/" + id + "/"
}
