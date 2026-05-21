package campaign

import (
	"fmt"
	"html"
	"math/rand"
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

func (c BirthdayCampaign) SelectTemplate(now time.Time) string {
	if len(c.TemplateIDs) == 0 {
		return ""
	}
	randomIntn := c.RandomIntn
	if randomIntn == nil {
		randomIntn = rand.New(rand.NewSource(now.UnixNano())).Intn
	}
	return c.TemplateIDs[randomIntn(len(c.TemplateIDs))]
}

func (c BirthdayCampaign) BuildMergeData(user domain.User, voucherCode string, wishlist []domain.WishlistItem, fyp []domain.FYPItem, popular []domain.FYPItem) map[string]any {
	fypItems := fyp
	if len(fypItems) == 0 {
		fypItems = popular
	}

	return map[string]any{
		"name":           user.Name,
		"voucher_code":   voucherCode,
		"wishlist_items": wishlist,
		"fyp_items":      fypItems,
		"wishlist_html":  RenderWishlistHTML(wishlist),
		"fyp_html":       RenderFYPHTML(fypItems),
		"action_url":     c.ActionURL,
		"closing":        c.Closing,
	}
}

func RenderWishlistHTML(items []domain.WishlistItem) string {
	if len(items) == 0 {
		return `<p style="margin:0;color:#6b7280;">Your wishlist is waiting for your next pick.</p>`
	}

	var builder strings.Builder
	builder.WriteString(`<ul style="margin:0;padding-left:18px;">`)
	for _, item := range items {
		name := html.EscapeString(item.Name)
		url := html.EscapeString(item.URL)
		if url == "" {
			builder.WriteString(fmt.Sprintf(`<li style="margin:0 0 8px 0;">%s</li>`, name))
			continue
		}
		builder.WriteString(fmt.Sprintf(`<li style="margin:0 0 8px 0;"><a href="%s" style="color:#2563eb;text-decoration:none;">%s</a></li>`, url, name))
	}
	builder.WriteString(`</ul>`)
	return builder.String()
}

func RenderFYPHTML(items []domain.FYPItem) string {
	if len(items) == 0 {
		return `<p style="margin:0;color:#6b7280;">Recommended picks are being refreshed for you.</p>`
	}

	var builder strings.Builder
	builder.WriteString(`<ul style="margin:0;padding-left:18px;">`)
	for _, item := range items {
		name := html.EscapeString(item.Name)
		kind := html.EscapeString(item.Kind)
		if kind == "" {
			kind = "Recommended"
		}
		builder.WriteString(fmt.Sprintf(`<li style="margin:0 0 8px 0;"><strong>%s</strong> <span style="color:#6b7280;">%s</span></li>`, name, kind))
	}
	builder.WriteString(`</ul>`)
	return builder.String()
}
