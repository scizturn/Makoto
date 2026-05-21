package campaign

import (
	"math/rand"
	"time"

	"github.com/kyou-id/makoto/internal/domain"
)

type BirthdayCampaign struct {
	TemplateIDs []string
	Closing     string
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
		"closing":        c.Closing,
	}
}
