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
		"cart_html":  RenderCartItemsHTMLWithURL(cartItems, c.CartURL),
		"reco_html":  RenderLeftoverRecoHTML(recoItems),
		"cart_url":   c.CartURL,
		"closing":    c.Closing,
	}
}

func RenderLeftoverRecoHTML(items []domain.FYPItem) string {
	if len(items) == 0 {
		return ""
	}

	theme := strings.TrimSpace(items[0].SeriesName)
	if theme == "" {
		theme = "koleksimu"
	} else {
		theme = theme + "-mu"
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf(`
<table role="presentation" width="100%%" cellspacing="0" cellpadding="0" style="width:100%%;border-collapse:collapse;">
  <tr>
    <td align="center" style="padding:0 0 20px;">
      <p style="margin:0 0 8px;color:#ff4b0a;font-size:13px;font-weight:900;letter-spacing:3px;text-transform:uppercase;">Pilihan Buat Nemenin</p>
      <h2 style="margin:0;color:#2b2b2b;font-size:28px;font-weight:900;line-height:1.25;">Tiga ini paling pas di rak %s</h2>
      <div style="display:inline-block;margin:20px 0 0;padding:8px 20px;background:#ffffff;border:1px solid #e5e7eb;border-radius:24px;box-shadow:0 4px 12px rgba(0,0,0,0.10);color:#3f3f46;font-size:14px;font-weight:900;">buat nemenin koleksimu</div>
    </td>
  </tr>
</table>`, html.EscapeString(theme)))

	for index, item := range items {
		if index >= 3 {
			break
		}
		builder.WriteString(renderLeftoverRecoCard(item, index))
	}

	return builder.String()
}

func renderLeftoverRecoCard(item domain.FYPItem, index int) string {
	name, version := cartItemDisplayName(item.Name, item.SeriesName)
	if version != "" {
		name = strings.TrimSpace(name + " " + version)
	}
	safeName := html.EscapeString(name)
	safeSeries := html.EscapeString(displayManufacturerOrFallback(item.SeriesName, "Kyou Pick"))
	safeImageURL := html.EscapeString(item.ImageURL)
	safeURL := html.EscapeString(itemURL(item.ID))

	statusLabel := "Pre-Order"
	status := strings.ToUpper(strings.TrimSpace(item.Status))
	if status == "READY" || status == "READY STOCK" {
		statusLabel = "Ready Stock"
	}

	cardBorder := "#e5e7eb"
	cardBg := "#ffffff"
	align := "left"
	imageFirst := true
	if index%2 == 1 {
		cardBorder = "#ffc9b5"
		cardBg = "#fff7f3"
		align = "right"
		imageFirst = false
	}

	imageHTML := fmt.Sprintf(`
<td width="150" valign="middle" style="width:150px;padding:%s;">
  <div style="position:relative;width:132px;height:132px;border-radius:10px;overflow:hidden;background:#f3f4f6;">
    <img src="%s" alt="%s" width="132" height="132" style="display:block;width:132px;height:132px;object-fit:cover;border:0;">
  </div>
  <div style="margin:-124px 0 94px 10px;width:78px;background:#64748b;color:#ffffff;font-size:12px;font-weight:900;line-height:24px;text-align:center;border-radius:4px;">%s</div>
</td>`, recoImagePadding(imageFirst), safeImageURL, safeName, html.EscapeString(statusLabel))
	if safeImageURL == "" {
		imageHTML = fmt.Sprintf(`
<td width="150" valign="middle" style="width:150px;padding:%s;">
  <div style="width:132px;height:132px;border-radius:10px;background:#f3f4f6;"></div>
</td>`, recoImagePadding(imageFirst))
	}

	textHTML := fmt.Sprintf(`
<td valign="middle" align="%s" style="padding:%s;">
  <p style="margin:0 0 5px;color:#a3a3a3;font-size:13px;font-weight:900;letter-spacing:1px;text-transform:uppercase;">%s</p>
  <h3 style="margin:0 0 8px;color:#2b2b2b;font-size:20px;font-weight:900;line-height:1.25;">%s<span style="color:#ff4b0a;">®</span></h3>
  <p style="margin:0 0 16px;color:#4b5563;font-size:15px;font-weight:700;line-height:1.55;">%s</p>
  <a href="%s" style="display:inline-block;background:#ff4b0a;border-bottom:4px solid #c93a00;border-radius:22px;color:#ffffff;font-size:14px;font-weight:900;letter-spacing:1px;line-height:42px;padding:0 28px;text-decoration:none;text-transform:uppercase;">Belanja Sekarang</a>
</td>`, align, recoTextPadding(imageFirst), safeSeries, safeName, leftoverRecoDescription(index), safeURL)

	firstCell, secondCell := imageHTML, textHTML
	if !imageFirst {
		firstCell, secondCell = textHTML, imageHTML
	}

	return fmt.Sprintf(`
<table role="presentation" width="100%%" cellspacing="0" cellpadding="0" style="width:100%%;border-collapse:separate;margin:0 0 16px;background:%s;border:1px solid %s;border-radius:14px;box-shadow:0 3px 12px rgba(0,0,0,0.08);overflow:hidden;">
  <tr>%s%s</tr>
</table>`, cardBg, cardBorder, firstCell, secondCell)
}

func recoImagePadding(imageFirst bool) string {
	if imageFirst {
		return "16px 0 16px 18px"
	}
	return "16px 18px 16px 0"
}

func recoTextPadding(imageFirst bool) string {
	if imageFirst {
		return "18px 22px 18px 20px"
	}
	return "18px 20px 18px 22px"
}

func leftoverRecoDescription(index int) string {
	descriptions := []string{
		"Sama-sama figure karakter, biar rak koleksimu makin rame.",
		"Satu kategori pilihan yang pas disandingin sama item incaranmu.",
		"Ready stock, langsung bisa dikirim nemenin koleksimu.",
	}
	if index < len(descriptions) {
		return descriptions[index]
	}
	return descriptions[0]
}

func RenderCartItemsHTML(items []domain.WishlistItem) string {
	return RenderCartItemsHTMLWithURL(items, "https://kyou.id/user/cart")
}

func RenderCartItemsHTMLWithURL(items []domain.WishlistItem, cartURL string) string {
	if len(items) == 0 {
		return `<p style="margin:0;color:#6b7280;">Keranjangmu lagi nunggu kamu balik nih!</p>`
	}
	if strings.TrimSpace(cartURL) == "" {
		cartURL = "https://kyou.id/user/cart"
	}

	totalItems := len(items)
	productCount := cartProductCount(items)

	var builder strings.Builder

	// Browser chrome wrapper
	builder.WriteString(`<table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="width:100%;border-collapse:collapse;font-family:'Nunito',Arial,Helvetica,sans-serif;border:1px solid #d6d6d6;border-radius:16px;overflow:hidden;background:#f1f1f1;">`)

	// Browser chrome top bar
	builder.WriteString(`
<tr>
  <td style="padding:16px 16px 18px;background:#eeeeef;border-bottom:1px solid #d6d6d6;">
    <table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="border-collapse:collapse;">
      <tr>
        <td width="48" valign="middle" style="padding:0 10px 0 0;white-space:nowrap;">
          <span style="display:inline-block;width:10px;height:10px;background:#cfcfcf;border-radius:50%;margin:0 6px 0 0;vertical-align:middle;"></span><!--
          --><span style="display:inline-block;width:10px;height:10px;background:#cfcfcf;border-radius:50%;margin:0 6px 0 0;vertical-align:middle;"></span><!--
          --><span style="display:inline-block;width:10px;height:10px;background:#cfcfcf;border-radius:50%;vertical-align:middle;"></span>
        </td>
        <td valign="middle" style="padding:0;">
          <div style="background:#ffffff;border:1px solid #e5e7eb;border-radius:18px;padding:12px 18px;display:block;width:518px;">
            <span style="font-size:14px;font-weight:900;color:#334155;">kyou.id/keranjang</span>
          </div>
        </td>
      </tr>
    </table>
  </td>
</tr>`)

	// Keranjang Saya header row
	builder.WriteString(fmt.Sprintf(`
<tr>
  <td style="padding:20px 20px 0;">
    <table role="presentation" width="100%%" cellspacing="0" cellpadding="0" style="width:100%%;border-collapse:collapse;margin-bottom:12px;">
      <tr>
        <td align="left" style="font-size:20px;font-weight:900;color:#1f2937;">Keranjang Saya</td>
        <td align="right" style="font-size:13px;font-weight:800;color:#7b7b7b;">%d item &bull; %d produk</td>
      </tr>
    </table>
  </td>
</tr>`, totalItems, productCount))

	// White card with items
	builder.WriteString(`
<tr>
  <td style="padding:0 20px 20px;">
    <table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="width:100%;border-collapse:separate;background-color:#ffffff;border:1px solid #e5e7eb;border-radius:12px;overflow:hidden;">`)

	// Pilih semua row
	builder.WriteString(`
<tr>
  <td style="padding:14px 16px;border-bottom:1px solid #e5e7eb;">
    <table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="width:100%;border-collapse:collapse;">
      <tr>
        <td width="28" valign="middle">
          <div style="width:22px;height:22px;background-color:#ea580c;border-radius:6px;text-align:center;line-height:22px;">
            <span style="color:white;font-size:15px;font-weight:bold;">&#10003;</span>
          </div>
        </td>
        <td valign="middle" style="padding-left:12px;font-size:15px;font-weight:800;color:#111827;">Pilih Semua</td>
        <td align="right" valign="middle" style="font-size:14px;font-weight:700;color:#9ca3af;">Hapus</td>
      </tr>
    </table>
  </td>
</tr>`)

	for i, item := range items {
		borderBottom := "1px solid #e5e7eb"
		if i == len(items)-1 {
			borderBottom = "none"
		}
		builder.WriteString(fmt.Sprintf(`<tr><td style="padding:16px;border-bottom:%s;">`, borderBottom))
		builder.WriteString(renderCartListItem(item))
		builder.WriteString(`</td></tr>`)
	}

	builder.WriteString(`</table></td></tr>`)
	builder.WriteString(renderCartButton(cartURL))
	builder.WriteString(`</table>`)

	return builder.String()
}

func renderCartButton(cartURL string) string {
	safeCartURL := html.EscapeString(cartURL)
	return fmt.Sprintf(`
<tr>
  <td style="padding:0 20px 22px;">
    <table role="presentation" width="100%%" cellspacing="0" cellpadding="0" style="width:100%%;border-collapse:separate;background:#ffffff;border:1px solid #e5e7eb;border-radius:12px;">
      <tr>
        <td style="padding:20px;">
          <a href="%s" style="display:block;width:100%%;background:#ff4b0a;border-bottom:5px solid #c93a00;border-radius:24px;color:#ffffff;font-size:16px;font-weight:900;line-height:48px;text-align:center;text-decoration:none;">Lanjut ke Keranjang</a>
        </td>
      </tr>
    </table>
  </td>
</tr>`, safeCartURL)
}

func renderCartListItem(item domain.WishlistItem) string {
	mainName, version := cartItemDisplayName(item.Name, item.SeriesName)
	safeName := html.EscapeString(mainName)
	safeVersion := html.EscapeString(version)
	safeSeries := html.EscapeString(displayManufacturerOrFallback(item.SeriesName, "Kyou Pick"))
	safeURL := html.EscapeString(item.URL)
	safeImageURL := html.EscapeString(item.ImageURL)
	priceFormatted := formatPrice(item.Price)

	imgHTML := fmt.Sprintf(`<img src="%s" alt="%s" width="80" height="80" style="display:block;width:80px;height:80px;object-fit:cover;background:#e5e7eb;border-radius:8px;border:1px solid #e5e7eb;">`, safeImageURL, safeName)
	if safeImageURL == "" {
		imgHTML = `<div style="width:80px;height:80px;background:#e5e7eb;border-radius:8px;border:1px solid #e5e7eb;"></div>`
	}

	versionP := ""
	if safeVersion != "" {
		versionP = fmt.Sprintf(` <span style="color:#6b7280;font-size:14px;font-weight:700;">%s</span>`, safeVersion)
	}

	statusBadge := ""
	status := strings.ToUpper(strings.TrimSpace(item.Status))
	if status == "READY" || status == "READY STOCK" {
		statusBadge = `<span style="display:inline-block;padding:4px 10px;border-radius:4px;background:#10b981;color:#ffffff;font-size:11px;font-weight:800;">Ready Stock</span>`
	} else if status == "FLASH PO" {
		statusBadge = `<span style="display:inline-block;padding:4px 10px;border-radius:4px;background:#f97316;color:#ffffff;font-size:11px;font-weight:800;">Flash PO</span>`
	} else {
		statusBadge = `<span style="display:inline-block;padding:4px 10px;border-radius:4px;background:#6b7280;color:#ffffff;font-size:11px;font-weight:800;">Pre-Order</span>`
	}

	inner := fmt.Sprintf(`
<table role="presentation" width="100%%" cellspacing="0" cellpadding="0" style="width:100%%;border-collapse:collapse;">
  <tr>
    <td width="28" valign="top" style="padding-top:28px;">
      <div style="width:22px;height:22px;background-color:#ea580c;border-radius:6px;text-align:center;line-height:22px;">
        <span style="color:white;font-size:15px;font-weight:bold;">&#10003;</span>
      </div>
    </td>
    <td width="80" valign="top" style="padding-left:16px;">
      %s
    </td>
    <td valign="top" style="padding-left:16px;padding-right:16px;">
      <p style="margin:0 0 4px;color:#9ca3af;font-size:13px;font-weight:800;">%s</p>
      <p style="margin:0 0 10px;color:#111827;font-size:16px;font-weight:800;line-height:1.4;">%s%s</p>
      <div style="margin:0;">
        %s
        <span style="display:inline-block;margin-left:10px;color:#10b981;font-size:13px;font-weight:800;">Stok tersedia</span>
      </div>
    </td>
    <td width="120" valign="top" align="right">
      <p style="margin:0 0 16px;color:#111827;font-size:16px;font-weight:900;">IDR %s</p>
      
      <table role="presentation" cellspacing="0" cellpadding="0" style="display:inline-table;">
        <tr>
          <td valign="middle" style="padding-right:12px;">
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none">
              <path d="M4 7h16M9 7V5a1 1 0 011-1h4a1 1 0 011 1v2m-9 0 1 13a1 1 0 001 1h6a1 1 0 001-1l1-13" stroke="#a2a2a2" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"></path>
            </svg>
          </td>
          <td valign="middle">
            <div style="border:1px solid #d1d5db;border-radius:8px;overflow:hidden;background:#ffffff;display:inline-block;">
              <table role="presentation" cellspacing="0" cellpadding="0" style="border-collapse:collapse;">
                <tr>
                  <td width="30" height="30" align="center" valign="middle" style="background:#f9fafb;color:#9ca3af;font-size:18px;font-weight:bold;border-right:1px solid #d1d5db;">&minus;</td>
                  <td width="40" height="30" align="center" valign="middle" style="font-size:15px;font-weight:900;color:#111827;">1</td>
                  <td width="30" height="30" align="center" valign="middle" style="background:#ffffff;color:#ea580c;font-size:18px;font-weight:bold;border-left:1px solid #d1d5db;">&#43;</td>
                </tr>
              </table>
            </div>
          </td>
        </tr>
      </table>

    </td>
  </tr>
</table>`, imgHTML, safeSeries, safeName, versionP, statusBadge, priceFormatted)

	if safeURL == "" {
		return inner
	}
	return fmt.Sprintf(`<a href="%s" style="display:block;text-decoration:none;color:inherit;">%s</a>`, safeURL, inner)
}

func cartItemDisplayName(name string, seriesName string) (string, string) {
	name = strings.TrimSpace(name)
	for strings.HasPrefix(name, "[") {
		end := strings.Index(name, "]")
		if end < 0 {
			break
		}
		name = strings.TrimSpace(name[end+1:])
	}
	if i := strings.LastIndex(name, " ("); i >= 0 {
		if suf := name[i:]; strings.HasSuffix(suf, ")") && strings.Contains(strings.ToLower(suf), "cm") {
			name = strings.TrimSpace(name[:i])
		}
	}
	name = stripItemNoise(name, seriesName)
	if displayName, version := splitItemVersion(name); strings.TrimSpace(version) != "" {
		return strings.TrimSpace(displayName), strings.TrimRight(strings.TrimSpace(version), ".")
	}
	return strings.TrimSpace(name), ""
}

func cartProductCount(items []domain.WishlistItem) int {
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		key := strings.TrimSpace(item.ID)
		if key == "" {
			key = strings.TrimSpace(item.Name)
		}
		if key == "" {
			continue
		}
		seen[key] = struct{}{}
	}
	if len(seen) == 0 {
		return len(items)
	}
	return len(seen)
}

func formatPrice(price int) string {
	s := strconv.Itoa(price)
	n := len(s)
	if n <= 3 {
		return s
	}
	var buf strings.Builder
	rem := n % 3
	if rem > 0 {
		buf.WriteString(s[:rem])
		if n > rem {
			buf.WriteString(".")
		}
	}
	for i := rem; i < n; i += 3 {
		buf.WriteString(s[i : i+3])
		if i+3 < n {
			buf.WriteString(".")
		}
	}
	return buf.String()
}
