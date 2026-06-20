package worker

import (
	"context"
	"testing"
	"time"

	"github.com/kyou-id/makoto/internal/campaign"
	"github.com/kyou-id/makoto/internal/domain"
)

func TestDiscountedWishlistProcessorSendsValidJob(t *testing.T) {
	sender := &fakeSender{}
	processor := NewDiscountedWishlistProcessor(sender, fakeValidator{valid: true}, campaign.DiscountedWishlistCampaign{
		TemplateIDs: []string{"discounted_wishlist1.html"},
		WishlistURL: "https://kyou.id/user/wishlist",
		RandomIntn:  func(int) int { return 0 },
	})
	processor.Renderer = fakeRenderer{subject: "Wishlist diskon", html: "<p>promo</p>"}

	result, err := processor.Process(context.Background(), discountedWishlistTestJob(true, "budi@example.test"))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Outcome != ProcessOutcomeSent || len(sender.messages) != 1 {
		t.Fatalf("expected sent outcome, result=%#v messages=%#v", result, sender.messages)
	}
}

func TestDiscountedWishlistProcessorSkipsInactiveUser(t *testing.T) {
	sender := &fakeSender{}
	processor := NewDiscountedWishlistProcessor(sender, fakeValidator{valid: true}, campaign.DiscountedWishlistCampaign{})

	result, err := processor.Process(context.Background(), discountedWishlistTestJob(false, "budi@example.test"))
	if err != nil || result.Outcome != ProcessOutcomeSkipped || result.SkipReason != "inactive_user" {
		t.Fatalf("unexpected result=%#v err=%v", result, err)
	}
	if len(sender.messages) != 0 {
		t.Fatalf("expected no send, got %#v", sender.messages)
	}
}

func TestDiscountedWishlistProcessorSkipsRejectedEmail(t *testing.T) {
	sender := &fakeSender{}
	processor := NewDiscountedWishlistProcessor(sender, fakeValidator{valid: false}, campaign.DiscountedWishlistCampaign{})

	result, err := processor.Process(context.Background(), discountedWishlistTestJob(true, "bad@example.test"))
	if err != nil || result.Outcome != ProcessOutcomeSkipped || result.SkipReason != "email_validation_rejected" {
		t.Fatalf("unexpected result=%#v err=%v", result, err)
	}
	if len(sender.messages) != 0 {
		t.Fatalf("expected no send, got %#v", sender.messages)
	}
}

func TestDiscountedWishlistProcessorReturnsRenderFailure(t *testing.T) {
	processor := NewDiscountedWishlistProcessor(&fakeSender{}, fakeValidator{valid: true}, campaign.DiscountedWishlistCampaign{
		TemplateIDs: []string{"discounted_wishlist1.html"},
		RandomIntn:  func(int) int { return 0 },
	})
	processor.Renderer = discountedWishlistErrorRenderer{}

	result, err := processor.Process(context.Background(), discountedWishlistTestJob(true, "budi@example.test"))
	if err == nil || result.Outcome != "" {
		t.Fatalf("expected render failure, result=%#v err=%v", result, err)
	}
}

func TestDiscountedWishlistProcessorReturnsProviderFailure(t *testing.T) {
	sender := &discountedWishlistErrorSender{}
	processor := NewDiscountedWishlistProcessor(sender, fakeValidator{valid: true}, campaign.DiscountedWishlistCampaign{
		TemplateIDs: []string{"discounted_wishlist1.html"},
		RandomIntn:  func(int) int { return 0 },
	})

	result, err := processor.Process(context.Background(), discountedWishlistTestJob(true, "budi@example.test"))
	if err == nil || result.Outcome != "" || sender.calls != 1 {
		t.Fatalf("expected provider failure, result=%#v calls=%d err=%v", result, sender.calls, err)
	}
}

func discountedWishlistTestJob(active bool, email string) domain.DiscountedWishlistJob {
	return domain.DiscountedWishlistJob{
		ID:      "discounted-wishlist-2026-06-20-user-123",
		UserID:  "123",
		Date:    time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC),
		Attempt: 1,
		User:    domain.User{ID: "123", Name: "Budi", Email: email, IsActive: active},
		Items: []domain.DiscountedWishlistItem{{
			ID: "1001", Name: "Figure", OriginalPrice: 100000, DiscountPrice: 80000, IsWishlisted: true,
		}},
	}
}

type discountedWishlistTestError string

func (e discountedWishlistTestError) Error() string { return string(e) }

type discountedWishlistErrorRenderer struct{}

func (discountedWishlistErrorRenderer) Render(string, map[string]any) (string, string, error) {
	return "", "", discountedWishlistTestError("render failed")
}

type discountedWishlistErrorSender struct {
	calls int
}

func (s *discountedWishlistErrorSender) SendTemplate(context.Context, domain.EmailMessage) (domain.SendResult, error) {
	s.calls++
	return domain.SendResult{StatusCode: 503, Response: "unavailable"}, discountedWishlistTestError("provider failed")
}
