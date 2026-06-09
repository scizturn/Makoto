package campaign

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/kyou-id/makoto/internal/domain"
)

type AnniversaryCampaign struct {
	TemplateIDs []string
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

func (c AnniversaryCampaign) BuildMergeData(user domain.User, voucherCode string, wishlist []domain.WishlistItem, years int, historicalItem domain.HistoricalItem) map[string]any {
	return map[string]any{
		"name":            user.Name,
		"voucher_code":    voucherCode,
		"years":           years,
		"historical_item": historicalItem,
		"wishlist_items":  wishlist,
		"wishlist_html":   RenderWishlistGridHTML(wishlist),
		"historical_html": RenderHistoricalItemHTML(historicalItem, years),
		"action_url":      actionURLWithClaim(c.ActionURL, voucherCode),
		"closing":         c.Closing,
	}
}

func RenderHistoricalItemHTML(item domain.HistoricalItem, years int) string {
	if item.Name == "" && item.ImageURL == "" {
		return ""
	}
	
	orderText := "belanjaan pertama"
	if years > 1 {
		orderText = fmt.Sprintf("belanjaan ke-%d", years)
	}

	return fmt.Sprintf(`
		<div style="margin:20px 0;padding:20px;border:1px solid #e5e7eb;border-radius:12px;background:#f9fafb;text-align:center;">
			<h3 style="margin:0 0 10px;color:#1f2937;font-size:18px;">Ingat ga ini %s kamu?</h3>
			<img src="%s" alt="%s" width="120" style="display:block;margin:0 auto 10px;border-radius:8px;">
			<p style="margin:0 0 5px;font-weight:bold;color:#111827;">%s</p>
			<p style="margin:0;color:#6b7280;font-size:14px;">%d hari yang lalu kamu membelinya.</p>
		</div>
	`, orderText, item.ImageURL, item.Name, item.Name, item.DaysAgo)
}
