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

var idMonths = [...]string{
	"", "Januari", "Februari", "Maret", "April", "Mei", "Juni",
	"Juli", "Agustus", "September", "Oktober", "November", "Desember",
}

func joinDateFormatted(date time.Time, years int) string {
	// anniversary filter matches DATE_FORMAT(created_at, '%m-%d') = today,
	// so join month+day == date month+day; only year differs.
	joinYear := date.Year() - years
	if joinYear < 2014 {
		joinYear = 2014
	}
	return fmt.Sprintf("%d %s %d", date.Day(), idMonths[date.Month()], joinYear)
}

type AnniversaryCampaign struct {
	TemplateIDs []string
	Subjects    []string
	Closing     string
	ActionURL   string
	RandomIntn  func(n int) int
}

func (c AnniversaryCampaign) SelectTemplate(now time.Time, key string) string {
	if len(c.TemplateIDs) == 0 {
		return ""
	}
	randomIntn := c.RandomIntn
	if randomIntn == nil {
		randomIntn = rand.New(rand.NewSource(templateSeed(now, key))).Intn
	}
	return c.TemplateIDs[randomIntn(len(c.TemplateIDs))]
}

func (c AnniversaryCampaign) SelectSubject(now time.Time, key string) string {
	if len(c.Subjects) == 0 {
		return ""
	}
	randomIntn := c.RandomIntn
	if randomIntn == nil {
		randomIntn = rand.New(rand.NewSource(templateSeed(now, key))).Intn
	}
	return c.Subjects[randomIntn(len(c.Subjects))]
}

type subjectData struct {
	Name      string
	FirstName string
	Years     int
}

func (c AnniversaryCampaign) RenderSubject(tpl string, user domain.User, years int) (string, error) {
	firstName := user.Name
	if i := strings.Index(user.Name, " "); i > 0 {
		firstName = user.Name[:i]
	}
	t, err := texttemplate.New("subject").Parse(tpl)
	if err != nil {
		return "", err
	}
	var buf strings.Builder
	if err := t.Execute(&buf, subjectData{Name: user.Name, FirstName: firstName, Years: years}); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func ordinalSuffix(n int) string {
	switch {
	case n%100 == 11 || n%100 == 12 || n%100 == 13:
		return "TH"
	case n%10 == 1:
		return "ST"
	case n%10 == 2:
		return "ND"
	case n%10 == 3:
		return "RD"
	default:
		return "TH"
	}
}

func (c AnniversaryCampaign) BuildMergeData(user domain.User, voucherCode string, wishlist []domain.WishlistItem, years int, historicalItem domain.HistoricalItem, date time.Time) map[string]any {
	firstName := user.Name
	if i := strings.Index(user.Name, " "); i > 0 {
		firstName = user.Name[:i]
	}
	return map[string]any{
		"name":                user.Name,
		"first_name":          firstName,
		"voucher_code":        voucherCode,
		"years":               years,
		"join_date":           joinDateFormatted(date, years),
		"anniversary_edition": fmt.Sprintf("%d%s ANNIVERSARY EDITION", years, ordinalSuffix(years)),
		"historical_item":     historicalItem,
		"wishlist_items":      wishlist,
		"wishlist_html":       RenderWishlistGridHTML(wishlist),
		"historical_html":     RenderHistoricalItemHTML(historicalItem, years),
		"action_url":          actionURLWithClaim(c.ActionURL, voucherCode),
		"closing":             c.Closing,
	}
}

var idMonthsShort = [...]string{
	"", "JAN", "FEB", "MAR", "APR", "MEI", "JUN",
	"JUL", "AGU", "SEP", "OKT", "NOV", "DES",
}

func formatDaysAgo(days int) string {
	if days >= 1000 {
		return fmt.Sprintf("%d.%03d", days/1000, days%1000)
	}
	return fmt.Sprintf("%d", days)
}

func RenderHistoricalItemHTML(item domain.HistoricalItem, years int) string {
	if item.Name == "" && item.ImageURL == "" {
		return ""
	}

	displayName := item.Name
	for strings.HasPrefix(displayName, "[") {
		end := strings.Index(displayName, "]")
		if end < 0 {
			break
		}
		displayName = strings.TrimSpace(displayName[end+1:])
	}
	safeName := html.EscapeString(displayName)
	safeImageURL := html.EscapeString(item.ImageURL)
	boldName := fmt.Sprintf(`<strong style="font-weight:800;color:#2c2422;">%s</strong>`, safeName)

	return fmt.Sprintf(`
<table role="presentation" width="560" cellspacing="0" cellpadding="0"
       style="width:560px;border-collapse:collapse;font-family:'Nunito',Arial,sans-serif;">
  <tr>
    <td valign="middle" style="padding:0 28px 0 0;">
      <div style="font-family:'Nunito',Arial,sans-serif;font-size:72px;font-weight:900;color:#2c2422;line-height:1;letter-spacing:-3px;">%s</div>
      <div style="font-family:'Nunito',Arial,sans-serif;font-size:20px;font-weight:800;color:#ff5a0a;letter-spacing:3.5px;margin:6px 0 8px;">HARI YANG LALU</div>
      <p style="margin:0;font-family:'Nunito',Arial,sans-serif;font-size:17px;font-weight:500;color:#565252;line-height:1.65;">kamu bawa pulang %s. Waktunya rakmu dapet temen baru.</p>
    </td>
    <td width="220" valign="middle" align="right" style="width:220px;padding:0;text-align:right;">
      <div style="display:inline-block;background:#ffffff;padding:8px 8px 28px 8px;box-shadow:0 6px 24px rgba(0,0,0,0.18),0 2px 6px rgba(0,0,0,0.10);transform:rotate(3deg);-webkit-transform:rotate(3deg);">
        <img src="%s" alt="%s" width="204" height="204"
             style="display:block;width:204px;height:204px;object-fit:cover;border:0;">
      </div>
    </td>
  </tr>
</table>
`, formatDaysAgo(item.DaysAgo), boldName, safeImageURL, safeName)
}
