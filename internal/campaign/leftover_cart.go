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

func (c LeftoverCartCampaign) BuildMergeData(user domain.User, cartItems []domain.WishlistItem, recoItems []domain.FYPItem, historicalItem domain.HistoricalItem, greeting string) map[string]any {
	firstName := user.Name
	if i := strings.Index(user.Name, " "); i > 0 {
		firstName = user.Name[:i]
	}

	return map[string]any{
		"name":            user.Name,
		"first_name":      firstName,
		"greeting":        greeting,
		"cart_items":      cartItems,
		"historical_item": historicalItem,
		"reco_items":      recoItems,
		"cart_html":       RenderCartItemsHTMLWithURL(cartItems, c.CartURL),
		"reco_html":       RenderLeftoverRecoHTML(recoItems, firstName, historicalItem),
		"footer_html":     RenderMemberversaryFooterHTML(c.Closing),
		"cart_url":        c.CartURL,
		"closing":         c.Closing,
	}
}

func RenderMemberversaryFooterHTML(closing string) string {
	return RenderMemberversaryFooterHTMLWithUnsubscribe(closing, "")
}

func RenderMemberversaryFooterHTMLWithUnsubscribe(closing, unsubscribeURL string) string {
	closing = strings.TrimSpace(closing)
	if closing == "" {
		closing = "Sampai ketemu lagi di Kyou!"
	}
	unsubscribeHTML := ""
	unsubscribeURL = strings.TrimSpace(unsubscribeURL)
	if parsed, err := url.ParseRequestURI(unsubscribeURL); err == nil && (parsed.Scheme == "https" || parsed.Scheme == "http") && parsed.Host != "" {
		unsubscribeHTML = fmt.Sprintf(`<p style="margin:12px 0 0;color:rgba(255,255,255,0.55);font-size:12px;line-height:1.5;">Tidak ingin menerima promo wishlist? <a href="%s" style="color:#ffffff;text-decoration:underline;">Atur preferensi email</a>.</p>`, html.EscapeString(unsubscribeURL))
	}
	footer := strings.Replace(memberversaryFooterHTML, "{{closing}}", html.EscapeString(closing), 1)
	return strings.Replace(footer, "{{unsubscribe}}", unsubscribeHTML, 1)
}

const memberversaryFooterHTML = `
            <!-- ── ABOUT KYOU IMAGE ── -->
            <table role="presentation" width="720" cellspacing="0" cellpadding="0"
                   style="width:720px;border-collapse:collapse;background:#2c2422;border-radius:20px 20px 0 0;overflow:hidden;">
            <tr><td style="padding:40px 52px 2px 52px;line-height:0;font-size:0;">
                <img src="https://kyoucdn.id/static/assets/about_kyou.jpg" alt="About Kyou" width="616"
                     style="display:block;width:616px;height:auto;border:0;border-radius:10px;" />
            </td></tr>
            </table>

            <!-- ── 2 CTA BUTTONS ── -->
            <table role="presentation" width="720" cellspacing="0" cellpadding="0"
                   style="width:720px;border-collapse:collapse;background:#2c2422;">
            <tr><td style="padding:32px 37px 32px;">
                <table role="presentation" width="646" cellspacing="0" cellpadding="0"
                       style="width:646px;border-collapse:collapse;">
                <tr>
                    <td width="50%" style="padding:0 4px 0 0;">
                        <a href="https://kyou.id/user/wishlist"
                           style="display:block;padding:20px 18px 18px;border:4px solid #5a351d;border-radius:999px;background:#ff5a0a;box-shadow:0 7px 0 #5a351d;color:#ffffff;font-size:16px;font-weight:900;letter-spacing:1px;line-height:1;text-align:center;text-decoration:none;">
                            <img src="https://kyoucdn.id/static/assets/k-mail.png" alt="" width="24" height="24"
                                 style="display:inline-block;width:24px;height:24px;margin:0 8px 0 0;border:0;vertical-align:middle;" />
                            <span style="display:inline-block;vertical-align:middle;">Cek wishlist kamu lainnya!</span>
                        </a>
                    </td>
                    <td width="50%" style="padding:0 0 0 4px;">
                        <a href="https://chat.whatsapp.com/E3kCtsnUfN70QYuMCtPgpM"
                           style="display:block;padding:20px 18px 18px;border:4px solid #5a351d;border-radius:999px;background:#2bd166;box-shadow:0 7px 0 #5a351d;color:#ffffff;font-size:16px;font-weight:900;letter-spacing:1px;line-height:1;text-align:center;text-decoration:none;">
                            <img src="https://kyoucdn.id/static/assets/wa-email.png" alt="" width="24" height="24"
                                 style="display:inline-block;width:24px;height:24px;margin:0 8px 0 0;border:0;vertical-align:middle;" />
                            <span style="display:inline-block;vertical-align:middle;">Join WA Kamar Wibu!</span>
                        </a>
                    </td>
                </tr>
                </table>
            </td></tr>
            </table>

            <!-- ── TOKO FISIK ── -->
            <table role="presentation" width="720" cellspacing="0" cellpadding="0"
                   style="width:720px;border-collapse:collapse;">
            <tr><td style="padding:36px;background:#2c2422;border-top:1px dashed #7a3b14;border-bottom:1px dashed #7a3b14;text-align:center;">
                <h2 style="margin:0 0 24px;color:#ffffff;font-size:34px;font-weight:900;line-height:1.15;">
                    Mampir ke Toko Fisik Kyou!
                </h2>
                <table role="presentation" width="648" cellspacing="0" cellpadding="0"
                       style="width:648px;border-collapse:collapse;">
                <tr>
                    <td width="25%" valign="top" style="width:25%;padding:0 7px 12px;">
                        <a href="https://kyou.id/stores/alpha" style="display:block;text-decoration:none;">
                            <img src="https://kyoucdn.id/static/assets/alpha.png" alt="Alpha Store" width="145"
                                 style="display:block;width:145px;height:auto;border:0;border-radius:10px;" />
                        </a>
                    </td>
                    <td width="25%" valign="top" style="width:25%;padding:0 7px 12px;">
                        <a href="https://kyou.id/stores/beta" style="display:block;text-decoration:none;">
                            <img src="https://kyoucdn.id/static/assets/beta.png" alt="Beta Store" width="145"
                                 style="display:block;width:145px;height:auto;border:0;border-radius:10px;" />
                        </a>
                    </td>
                    <td width="25%" valign="top" style="width:25%;padding:0 7px 12px;">
                        <a href="https://kyou.id/stores/gamma" style="display:block;text-decoration:none;">
                            <img src="https://kyoucdn.id/static/assets/gamma.png" alt="Gamma Store" width="145"
                                 style="display:block;width:145px;height:auto;border:0;border-radius:10px;" />
                        </a>
                    </td>
                    <td width="25%" valign="top" style="width:25%;padding:0 7px 12px;">
                        <a href="https://maps.app.goo.gl/kwAtN6H3Kf7cVSA49" style="display:block;text-decoration:none;">
                            <img src="https://kyoucdn.id/static/assets/delta.png" alt="Delta Store" width="145"
                                 style="display:block;width:145px;height:auto;border:0;border-radius:10px;" />
                        </a>
                    </td>
                </tr>
                </table>
            </td></tr>
            </table>

            <!-- ── SOCIAL MEDIA ── -->
            <table role="presentation" width="720" cellspacing="0" cellpadding="0"
                   style="width:720px;border-collapse:collapse;">
            <tr><td style="padding:18px 36px 46px;background:#2c2422;border-top:1px dashed #7a3b14;border-bottom:1px dashed #7a3b14;text-align:center;">
                <h2 style="margin:0 0 32px;color:#ffffff;font-size:40px;font-weight:900;line-height:1.15;">
                    Interact with Kyou!
                </h2>
                <table role="presentation" cellspacing="0" cellpadding="0" align="center"
                       style="margin:0 auto;border-collapse:collapse;">
                <tr>
                    <td style="padding:0 8px;"><a href="https://api.whatsapp.com/send/?phone=6281770092014&amp;text=Halo+Kyou"><img src="https://kyoucdn.id/static/assets/whatsapp.jpg" alt="WhatsApp" width="58" height="58" style="display:block;width:58px;height:58px;border:0;border-radius:999px;" /></a></td>
                    <td style="padding:0 8px;"><a href="https://www.facebook.com/kyouhobbyshop"><img src="https://kyoucdn.id/static/assets/facebook.jpg" alt="Facebook" width="58" height="58" style="display:block;width:58px;height:58px;border:0;border-radius:999px;" /></a></td>
                    <td style="padding:0 8px;"><a href="https://www.messenger.com/t/1466845816917156/"><img src="https://kyoucdn.id/static/assets/messenger.jpg" alt="Messenger" width="58" height="58" style="display:block;width:58px;height:58px;border:0;border-radius:999px;" /></a></td>
                    <td style="padding:0 8px;"><a href="https://www.youtube.com/@ChannelWibunyaKyou"><img src="https://kyoucdn.id/static/assets/youtube.jpg" alt="YouTube" width="58" height="58" style="display:block;width:58px;height:58px;border:0;border-radius:999px;" /></a></td>
                    <td style="padding:0 8px;"><a href="https://www.tiktok.com/@kyou.id?lang=en"><img src="https://kyoucdn.id/static/assets/tiktok.jpg" alt="TikTok" width="58" height="58" style="display:block;width:58px;height:58px;border:0;border-radius:999px;" /></a></td>
                    <td style="padding:0 8px;"><a href="https://kyou.id"><img src="https://kyoucdn.id/static/assets/kyou.jpg" alt="Kyou" width="58" height="58" style="display:block;width:58px;height:58px;border:0;border-radius:999px;" /></a></td>
                </tr>
                </table>
            </td></tr>
            </table>

            <!-- ── DARK CLOSING ── -->
            <table role="presentation" width="720" cellspacing="0" cellpadding="0"
                   style="width:720px;border-collapse:collapse;background:#2c2422;">
            <tr><td style="padding:36px;border-top:1px dashed #7a3b14;border-bottom:1px dashed #7a3b14;text-align:center;">
                <img src="https://kyoucdn.id/static/assets/brand_logo.png" alt="Kyou.id" width="132"
                     style="display:block;width:132px;height:auto;margin:0 auto 28px;border:0;" />
                <p style="margin:0 0 8px;color:#ffffff;font-size:20px;font-weight:900;line-height:1.2;">{{closing}}</p>
                <p style="margin:0 0 0;color:rgba(255,255,255,0.5);font-size:14px;font-weight:500;line-height:1.4;">
                    ©2014–2026 Kyou Hobby Shop
                </p>
                <p style="margin:0;color:rgba(255,255,255,0.4);font-size:14px;font-weight:500;line-height:1.55;">
                    Kamu menerima email ini karena terdaftar sebagai Teman Kyou.
                </p>
                {{unsubscribe}}
            </td></tr>
            </table>
`

func RenderLeftoverRecoHTML(items []domain.FYPItem, firstName string, historicalItem domain.HistoricalItem) string {
	if len(items) == 0 {
		return ""
	}
	firstName = strings.TrimSpace(firstName)
	if firstName == "" {
		firstName = "Teman Kyou"
	}

	theme := strings.TrimSpace(items[0].SeriesName)
	if theme == "" {
		theme = "koleksimu"
	}
	safeFirstName := html.EscapeString(firstName)
	safeTheme := html.EscapeString(theme)
	safeSeriesSearchURL := html.EscapeString(leftoverSeriesSearchURL(theme))
	characterName := leftoverHistoricalCharacterName(historicalItem.Name, theme)
	safeCharacterName := html.EscapeString(characterName)
	safeCharacterSearchURL := html.EscapeString(leftoverCharacterSearchURL(characterName))

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf(`
<table role="presentation" width="100%%" cellspacing="0" cellpadding="0" style="width:100%%;border-collapse:collapse;background:#f4f4f5;border:1px solid #d6d6d6;border-radius:16px;overflow:hidden;">
  <tr>
    <td style="padding:18px 22px;background:#0ea5d8;">
      <table role="presentation" width="100%%" cellspacing="0" cellpadding="0" style="width:100%%;border-collapse:collapse;">
        <tr>
          <td width="58" valign="middle" style="width:58px;">
            <div style="width:46px;height:46px;border-radius:50%%;background:#fff3e0;border:3px solid #ffffff;text-align:center;line-height:46px;overflow:hidden;">
              <img src="https://kyou.id/static/img/icon/kouka_chibi.png.webp" alt="Kyou.id" width="46" style="display:block;width:46px;height:46px;object-fit:cover;border:0;">
            </div>
          </td>
          <td valign="middle">
            <p style="margin:0 0 3px;color:#ffffff;font-size:18px;font-weight:900;line-height:1.1;">Kyou.id <span style="display:inline-block;margin-left:8px;padding:4px 10px;border-radius:14px;background:#7dd3fc;color:#ffffff;font-size:12px;font-weight:900;vertical-align:middle;">KOUKA</span></p>
            <p style="margin:0;color:#ffffff;font-size:13px;font-weight:900;"><span style="color:#22c55e;">●</span> Online</p>
          </td>
          <td width="24" align="right" valign="top" style="width:24px;color:#ffffff;font-size:28px;font-weight:900;line-height:24px;">−</td>
        </tr>
      </table>
    </td>
  </tr>
  <tr>
    <td style="padding:18px 16px 20px;background-image:url('https://images2.imgbox.com/11/6f/MewvOpFD_o.png');background-size:cover;background-position:center top;background-repeat:no-repeat;background-color:rgba(255,255,255,0.5);background-blend-mode:lighten;">
      <div style="margin:0 auto 14px;width:72px;border-radius:14px;background:#e5e5e5;color:#7a7a7a;font-size:12px;font-weight:900;line-height:26px;text-align:center;">Hari ini</div>
      <table role="presentation" cellspacing="0" cellpadding="0" style="border-collapse:collapse;margin:0 0 12px 58px;">
        <tr><td style="padding:13px 16px;background:#ffffff;border-radius:18px;box-shadow:0 1px 3px rgba(0,0,0,0.08);color:#2b2b2b;font-size:15px;font-weight:700;line-height:1.45;">Hai, <strong>%s</strong>! Kouka di sini.</td></tr>
      </table>
      <table role="presentation" cellspacing="0" cellpadding="0" style="border-collapse:collapse;margin:0 0 12px 58px;">
        <tr><td style="padding:13px 16px;background:#ffffff;border-radius:18px;box-shadow:0 1px 3px rgba(0,0,0,0.08);color:#2b2b2b;font-size:15px;font-weight:700;line-height:1.6;">Masih inget sama koleksimu yang ini?</td></tr>
      </table>
`, safeFirstName))
	builder.WriteString(renderLeftoverHistoricalCard(historicalItem))
	builder.WriteString(`
      <table role="presentation" cellspacing="0" cellpadding="0" style="border-collapse:collapse;margin:0 0 12px 58px;">
        <tr><td style="padding:13px 16px;background:#ffffff;border-radius:18px;box-shadow:0 1px 3px rgba(0,0,0,0.08);color:#2b2b2b;font-size:15px;font-weight:700;line-height:1.55;">Pilihan kamu keren banget!</td></tr>
      </table>
      <table role="presentation" cellspacing="0" cellpadding="0" style="border-collapse:collapse;margin:0 0 12px 58px;">
        <tr><td style="padding:13px 16px;background:#ffffff;border-radius:18px;box-shadow:0 1px 3px rgba(0,0,0,0.08);color:#2b2b2b;font-size:15px;font-weight:700;line-height:1.55;">Kouka udah intip-intip koleksimu, dan ini beberapa rekomendasi item yang cocok buat menemaninya:</td></tr>
      </table>`)

	for index, item := range items {
		if index >= 3 {
			break
		}
		builder.WriteString(renderLeftoverChatRecoItem(item))
	}

	builder.WriteString(`
      <table role="presentation" cellspacing="0" cellpadding="0" style="border-collapse:collapse;margin:4px 0 12px 58px;">
        <tr><td style="padding:13px 16px;background:#ffffff;border-radius:18px;box-shadow:0 1px 3px rgba(0,0,0,0.08);color:#2b2b2b;font-size:15px;font-weight:700;line-height:1.55;">Kouka tunggu kabar baiknya ya, semoga makin happy sama koleksinya!</td></tr>
      </table>
      <table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="width:100%;border-collapse:collapse;margin:2px 0 12px;">
        <tr>
          <td width="58" valign="middle" style="width:58px;">` + koukaAvatarHTML() + `</td>
          <td valign="middle">
            <div style="display:inline-block;padding:8px 16px;border-radius:18px;background:#ffffff;box-shadow:0 1px 3px rgba(0,0,0,0.08);">
              <span style="display:inline-block;width:7px;height:7px;border-radius:50%;background:#cfcfcf;margin-right:5px;"></span>
              <span style="display:inline-block;width:7px;height:7px;border-radius:50%;background:#cfcfcf;margin-right:5px;"></span>
              <span style="display:inline-block;width:7px;height:7px;border-radius:50%;background:#cfcfcf;"></span>
            </div>
          </td>
        </tr>
      </table>
      <table role="presentation" cellspacing="0" cellpadding="0" style="border-collapse:collapse;margin:0 0 0 58px;">
        <tr>
          <td style="padding:0 4px;"><a href="https://kyou.id/search?page=1%2C40&amp;sort=kyou_search_score&amp;sold=false" style="display:inline-block;padding:0 14px;border:1px solid #ff4b0a;border-radius:20px;color:#ff4b0a;font-size:13px;font-weight:900;line-height:36px;background:white;text-decoration:none;">Rekomendasi lainnya</a></td>
          <td style="padding:0 4px;"><a href="` + safeSeriesSearchURL + `" style="display:inline-block;padding:0 14px;border:1px solid #ff4b0a;border-radius:20px;color:#ff4b0a;font-size:13px;font-weight:900;line-height:36px;background:white;text-decoration:none;">` + safeTheme + `</a></td>
          <td style="padding:0 4px;"><a href="` + safeCharacterSearchURL + `" style="display:inline-block;padding:0 14px;border:1px solid #ff4b0a;border-radius:20px;color:#ff4b0a;font-size:13px;font-weight:900;line-height:36px;background:white;text-decoration:none;">` + safeCharacterName + `</a></td>
        </tr>
      </table>
    </td>
  </tr>
</table>`)
	return builder.String()
}

func leftoverSeriesSearchURL(seriesName string) string {
	seriesName = strings.TrimSpace(seriesName)
	if seriesName == "" {
		return "https://kyou.id/search?page=1%2C40&sort=kyou_search_score&sold=false"
	}
	return "https://kyou.id/search?page=1%2C40&series=" + url.QueryEscape(seriesName) + "&sort=kyou_search_score&sold=false"
}

func leftoverCharacterSearchURL(characterName string) string {
	characterName = strings.TrimSpace(characterName)
	if characterName == "" {
		return "https://kyou.id/search?page=1%2C40&sort=kyou_search_score&sold=false"
	}
	return "https://kyou.id/search?q=" + strings.ToLower(url.QueryEscape(characterName)) + "&page=1%2C40&sort=kyou_search_score&sold=false"
}

func leftoverHistoricalCharacterName(itemName string, seriesName string) string {
	name, version := cartItemDisplayName(itemName, seriesName)
	if version != "" {
		name = strings.TrimSpace(name + " " + version)
	}
	name = stripLeftoverCharacterPrefix(name)
	words := strings.Fields(name)
	for index, word := range words {
		normalized := strings.Trim(strings.ToLower(word), ":-_/")
		if leftoverProductTypeWords[normalized] && index+1 < len(words) {
			before := strings.Join(words[:index], " ")
			after := strings.Join(words[index+1:], " ")
			if strings.TrimSpace(before) == "" || leftoverGenericProductPrefix(before) {
				return after
			}
			return before
		}
		if leftoverProductTypeWords[normalized] && index > 0 {
			return strings.Join(words[:index], " ")
		}
	}
	return strings.TrimSpace(name)
}

func stripLeftoverCharacterPrefix(name string) string {
	prefixes := []string{
		"hololive English ",
		"hololive Indonesia ",
		"hololive Japan ",
		"hololive ",
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(strings.ToLower(name), strings.ToLower(prefix)) {
			return strings.TrimSpace(name[len(prefix):])
		}
	}
	return name
}

func leftoverGenericProductPrefix(text string) bool {
	words := strings.Fields(strings.ToLower(text))
	if len(words) == 0 {
		return true
	}
	for _, word := range words {
		word = strings.Trim(word, ":-_/")
		if !leftoverGenericProductWords[word] {
			return false
		}
	}
	return true
}

var leftoverProductTypeWords = map[string]bool{
	"acrylic":    true,
	"badge":      true,
	"charm":      true,
	"figure":     true,
	"keychain":   true,
	"nendoroid":  true,
	"plush":      true,
	"plushie":    true,
	"pouch":      true,
	"stand":      true,
	"tapestry":   true,
	"wallscroll": true,
}

var leftoverGenericProductWords = map[string]bool{
	"big":     true,
	"chibi":   true,
	"cute":    true,
	"desktop": true,
	"kuripan": true,
	"premium": true,
}

func koukaAvatarHTML() string {
	return `<div style="width:42px;height:42px;border-radius:50%;background:#fff3e0;border:3px solid #ff4b0a;text-align:center;line-height:42px;overflow:hidden;">
  <img src="https://kyou.id/static/img/icon/kouka_chibi.png.webp" alt="Kouka" width="42" style="display:block;width:42px;height:42px;object-fit:cover;border:0;">
</div>`
}

func renderLeftoverHistoricalCard(item domain.HistoricalItem) string {
	name := strings.TrimSpace(item.Name)
	if name == "" {
		return ""
	}
	displayName, version := cartItemDisplayName(name, "")
	if version != "" {
		displayName = strings.TrimSpace(displayName + " " + version)
	}
	safeName := html.EscapeString(displayName)
	safeImageURL := html.EscapeString(item.ImageURL)
	imageHTML := fmt.Sprintf(`<img src="%s" alt="%s" width="72" height="72" style="display:block;width:72px;height:72px;object-fit:cover;border:0;border-radius:6px;">`, safeImageURL, safeName)
	if safeImageURL == "" {
		imageHTML = `<div style="width:72px;height:72px;border-radius:6px;background:#f3f4f6;"></div>`
	}

	return fmt.Sprintf(`
      <table role="presentation" width="540" cellspacing="0" cellpadding="0" style="width:540px;border-collapse:separate;margin:0 0 12px 58px;background:#ffffff;border:1px solid #b7e4c7;border-radius:14px;box-shadow:0 2px 5px rgba(0,0,0,0.08);overflow:hidden;">
        <tr><td colspan="2" style="padding:8px 14px;background:#e8f8ef;border-bottom:1px solid #b7e4c7;color:#22a65a;font-size:12px;font-weight:900;letter-spacing:.6px;text-transform:uppercase;">◇ Koleksi Kamu</td></tr>
        <tr>
          <td width="86" valign="middle" style="width:86px;padding:12px 0 12px 12px;">%s</td>
          <td valign="middle" style="padding:12px 14px;">
            <p style="margin:0 0 3px;color:#a3a3a3;font-size:12px;font-weight:900;">Koleksi sebelumnya</p>
            <p style="margin:0 0 7px;color:#2b2b2b;font-size:16px;font-weight:900;line-height:1.25;">%s</p>
            <span style="display:inline-block;padding:4px 10px;background:#22c55e;color:#ffffff;border-radius:7px;font-size:12px;font-weight:900;">Koleksi kamu</span>
            <span style="margin-left:8px;color:#3f3f46;font-size:13px;font-weight:900;">Udah di rakmu</span>
          </td>
        </tr>
      </table>`, imageHTML, safeName)
}

func renderLeftoverChatRecoItem(item domain.FYPItem) string {
	name, version := cartItemDisplayName(item.Name, item.SeriesName)
	if version != "" {
		name = strings.TrimSpace(name + " " + version)
	}
	safeName := html.EscapeString(name)
	safeSeries := html.EscapeString(displayManufacturerOrFallback(item.SeriesName, "Kyou Pick"))
	safeImageURL := html.EscapeString(item.ImageURL)
	safeURL := html.EscapeString(itemURL(item.ID))
	priceFormatted := formatPrice(item.Price)

	statusLabel := "Pre-Order"
	statusBg := "#64748b"
	status := strings.ToUpper(strings.TrimSpace(item.Status))
	if status == "READY" || status == "READY STOCK" {
		statusLabel = "Ready Stock"
		statusBg = "#22c55e"
	}

	imageHTML := fmt.Sprintf(`<img src="%s" alt="%s" width="72" height="72" style="display:block;width:72px;height:72px;object-fit:cover;border:0;border-radius:6px;">`, safeImageURL, safeName)
	if safeImageURL == "" {
		imageHTML = `<div style="width:72px;height:72px;border-radius:6px;background:#f3f4f6;"></div>`
	}

	return fmt.Sprintf(`
      <table role="presentation" width="540" cellspacing="0" cellpadding="0" style="width:540px;border-collapse:separate;margin:0 0 8px 58px;background:#ffffff;border:1px solid #f1c9bb;border-radius:14px;box-shadow:0 2px 5px rgba(0,0,0,0.08);overflow:hidden;">
        <tr><td colspan="3" style="padding:8px 14px;background:#fff1ec;border-bottom:1px solid #f1c9bb;color:#ff4b0a;font-size:12px;font-weight:900;letter-spacing:.6px;text-transform:uppercase;">◇ Produk - Kyou Hobby Shop</td></tr>
        <tr>
          <td width="86" valign="middle" style="width:86px;padding:12px 0 12px 12px;">%s</td>
          <td valign="middle" style="padding:12px 10px;">
            <p style="margin:0 0 4px;color:#a3a3a3;font-size:12px;font-weight:900;text-transform:uppercase;">%s</p>
            <p style="margin:0 0 7px;color:#2b2b2b;font-size:16px;font-weight:900;line-height:1.25;">%s</p>
            <span style="display:inline-block;margin-right:10px;padding:4px 10px;background:%s;color:#ffffff;border-radius:6px;font-size:12px;font-weight:900;">%s</span>
            <span style="color:#2b2b2b;font-size:14px;font-weight:900;">IDR %s</span>
          </td>
          <td width="34" valign="middle" align="center" style="width:34px;"><a href="%s" style="display:block;color:#ff4b0a;font-size:32px;font-weight:900;text-decoration:none;">›</a></td>
        </tr>
        <tr><td colspan="3" align="center" style="padding:10px 0;border-top:1px solid #e5e5e5;"><a href="%s" style="color:#ff4b0a;font-size:13px;font-weight:900;text-decoration:none;">Lihat produk ›</a></td></tr>
      </table>
`, imageHTML, safeSeries, safeName, statusBg, html.EscapeString(statusLabel), priceFormatted, safeURL, safeURL)
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
            <span style="font-size:14px;font-weight:900;color:#334155;"><a href="https://kyou.id/user/cart" style="text-decoration:none;color:#334155;">https://kyou.id/user/cart</a></span>
          </div>
        </td>
      </tr>
    </table>
  </td>
</tr>`)

	// Shopping Cart header row + white card wrapped in a single link
	safeCartURLSection := html.EscapeString(cartURL)
	builder.WriteString(fmt.Sprintf(`
<tr>
  <td style="padding:20px 20px 0;">
    <table role="presentation" width="100%%" cellspacing="0" cellpadding="0" style="width:100%%;border-collapse:collapse;margin-bottom:12px;">
      <tr>
        <td align="left" style="font-size:20px;font-weight:900;color:#1f2937;">Shopping Cart</td>
        <td align="right" style="font-size:13px;font-weight:800;color:#7b7b7b;">%d item &bull; %d produk</td>
      </tr>
    </table>
  </td>
</tr>`, totalItems, productCount))

	// White card with items — entire card is wrapped in a link
	builder.WriteString(fmt.Sprintf(`
<tr>
  <td style="padding:0 20px 20px;">
    <a href="%s" style="display:block;text-decoration:none;color:inherit;">
    <table role="presentation" width="100%%" cellspacing="0" cellpadding="0" style="width:100%%;border-collapse:separate;background-color:#ffffff;border:1px solid #e5e7eb;border-radius:12px;overflow:hidden;">`, safeCartURLSection))

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

	builder.WriteString(`</table></a></td></tr>`)
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
