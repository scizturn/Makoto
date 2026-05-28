package campaign

import (
	"fmt"
	"hash/fnv"
	"html"
	"math/rand"
	"net/url"
	"strings"
	"time"

	"github.com/kyou-id/makoto/internal/domain"
)

type BirthdayCampaign struct {
	TemplateIDs []string
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

func renderProductCard(name string, seriesName string, fallbackLabel string, url string, imageURL string) string {
	displayName, version := splitItemDisplayName(name, seriesName)
	safeName := html.EscapeString(displayName)
	safeVersion := html.EscapeString(version)
	safeSeries := html.EscapeString(displayManufacturerOrFallback(seriesName, fallbackLabel))
	safeURL := html.EscapeString(url)
	safeImageURL := html.EscapeString(imageURL)
	imageHTML := `<div style="display:block;margin:auto;width:180px;height:180px;border-radius:4px;background:#f3f4f6;"></div>`
	if safeImageURL != "" {
		imageHTML = fmt.Sprintf(`<img src="%s" alt="%s" width="180" height="180" style="display:block;margin:auto;width:180px;border-radius:4px;height:180px;object-fit:cover;background:#f3f4f6;border:0;">`, safeImageURL, safeName)
	}

	nameStyle := `margin:0 0 48px 0;color:#0f172a;font-size:17px;font-weight:900;line-height:1.32;white-space:normal;word-break:break-word;`
	displayText := safeName
	if safeVersion != "" {
		displayText = fmt.Sprintf(`%s<br>%s`, safeName, safeVersion)
	}
	nameHTML := fmt.Sprintf(`<p style="%s">%s</p>`, nameStyle, displayText)

	cardHTML := fmt.Sprintf(`<div style="overflow:hidden;width:180px;height:360px;margin:auto;padding:12px;border:1px solid #9ca3af;border-radius:12px;background:#ffe0cf url('https://kyoucdn.id/static/assets/item_bg.jpg') center/cover no-repeat;text-align:left;">%s<div style="padding:12px 8px 4px;"><p style="margin:0 0 4px 0;overflow:hidden;color:#2f2b28;font-size:12px;font-weight:600;text-overflow:ellipsis;white-space:nowrap;">%s</p>%s</div></div>`, imageHTML, safeSeries, nameHTML)
	if safeURL == "" {
		return cardHTML
	}
	return fmt.Sprintf(`<a href="%s" style="display:block;width:230px;color:inherit;text-decoration:none;">%s</a>`, safeURL, cardHTML)
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
