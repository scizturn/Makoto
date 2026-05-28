package worker

import (
	"context"
	"testing"
	"time"

	"github.com/kyou-id/makoto/internal/campaign"
	"github.com/kyou-id/makoto/internal/domain"
)

func TestProcessorSkipsInactiveUser(t *testing.T) {
	repo := &fakeRepository{
		user: domain.User{ID: "1", Name: "Ruby", Email: "ruby@example.test", IsActive: false},
	}
	sender := &fakeSender{}
	processor := NewProcessor(repo, sender, fakeValidator{valid: true}, &fakeVoucherIssuer{}, campaign.BirthdayCampaign{})

	_, err := processor.Process(context.Background(), domain.BirthdayJob{UserID: "1", Date: time.Now()})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(sender.messages) != 0 {
		t.Fatalf("expected no sends, got %d", len(sender.messages))
	}
}

func TestProcessorSendsBirthdayTemplateMessage(t *testing.T) {
	repo := &fakeRepository{
		user:     domain.User{ID: "1", Name: "Ruby", Email: "ruby@example.test", IsActive: true},
		wishlist: []domain.WishlistItem{{ID: "wish-1", Name: "Figure Ruby"}},
		popular:  []domain.FYPItem{{ID: "popular-1", Name: "Popular Series", Kind: "series"}},
	}
	sender := &fakeSender{}
	issuer := &fakeVoucherIssuer{code: "HBD-RUBY-7K2M"}
	processor := NewProcessor(repo, sender, fakeValidator{valid: true}, issuer, campaign.BirthdayCampaign{
		TemplateIDs: []string{"tpl_001", "tpl_002", "tpl_003"},
		Closing:     "Selamat merayakan hari spesialmu di Kyou!",
		RandomIntn:  func(int) int { return 1 },
	})
	processor.Domain = "kyou.id"
	processor.FromEmail = "nandayo@kyou.id"
	processor.FromName = "Kyou.id"

	_, err := processor.Process(context.Background(), domain.BirthdayJob{
		UserID: "1",
		Date:   time.Date(2026, 5, 21, 7, 0, 0, 0, time.UTC),
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(sender.messages) != 1 {
		t.Fatalf("expected one send, got %d", len(sender.messages))
	}
	msg := sender.messages[0]
	if msg.ToEmail != "ruby@example.test" {
		t.Fatalf("expected recipient, got %q", msg.ToEmail)
	}
	if msg.Domain != "kyou.id" || msg.FromEmail != "nandayo@kyou.id" || msg.FromName != "Kyou.id" {
		t.Fatalf("unexpected sender fields: %#v", msg)
	}
	if msg.TemplateID != "tpl_002" {
		t.Fatalf("expected randomized template tpl_002, got %q", msg.TemplateID)
	}
	if msg.SubstitutionData["voucher_code"] != "HBD-RUBY-7K2M" {
		t.Fatalf("expected voucher merge data, got %#v", msg.SubstitutionData)
	}
	if issuer.requests != 1 {
		t.Fatalf("expected one voucher issue request, got %d", issuer.requests)
	}
}

func TestProcessorSendsFromFullRedisJobWithoutDatabaseStore(t *testing.T) {
	sender := &fakeSender{}
	issuer := &fakeVoucherIssuer{code: "HBD-GARVIN-7K2M"}
	processor := NewProcessor(nil, sender, fakeValidator{valid: true}, issuer, campaign.BirthdayCampaign{
		TemplateIDs: []string{"tpl_001", "tpl_002", "tpl_003"},
		Closing:     "Selamat merayakan hari spesialmu di Kyou!",
		ActionURL:   "https://kyou.id/user/my-voucher",
		RandomIntn:  func(int) int { return 2 },
	})
	processor.Domain = "kyou.id"
	processor.FromEmail = "nandayo@kyou.id"
	processor.FromName = "Kyou.id"

	_, err := processor.Process(context.Background(), domain.BirthdayJob{
		ID:     "birthday-2026-05-21-user-123",
		UserID: "123",
		Date:   time.Date(2026, 5, 21, 7, 0, 0, 0, time.UTC),
		User: domain.User{
			ID:       "123",
			Name:     "Garvin",
			Email:    "garvin@example.test",
			IsActive: true,
		},
		WishlistItems: []domain.WishlistItem{{ID: "wish-1", Name: "Figure"}},
		FYPItems:      []domain.FYPItem{{ID: "fyp-1", Name: "Chara", Kind: "character"}},
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(sender.messages) != 1 {
		t.Fatalf("expected one send, got %d", len(sender.messages))
	}
	msg := sender.messages[0]
	if msg.TemplateID != "tpl_003" {
		t.Fatalf("expected randomized template tpl_003, got %q", msg.TemplateID)
	}
	if msg.SubstitutionData["voucher_code"] != "HBD-GARVIN-7K2M" {
		t.Fatalf("expected generated voucher in merge data, got %#v", msg.SubstitutionData)
	}
	if msg.SubstitutionData["action_url"] != "https://kyou.id/user/my-voucher?claim=HBD-GARVIN-7K2M" {
		t.Fatalf("expected action_url merge data, got %#v", msg.SubstitutionData)
	}
	if msg.SubstitutionData["wishlist_html"] == "" || msg.SubstitutionData["fyp_html"] == "" {
		t.Fatalf("expected html merge data, got %#v", msg.SubstitutionData)
	}
}

func TestProcessorUsesVoucherCodeFromRedisJob(t *testing.T) {
	sender := &fakeSender{}
	issuer := &fakeVoucherIssuer{code: "SHOULD-NOT-BE-USED"}
	processor := NewProcessor(nil, sender, fakeValidator{valid: true}, issuer, campaign.BirthdayCampaign{
		TemplateIDs: []string{"tpl_001"},
		RandomIntn:  func(int) int { return 0 },
	})

	_, err := processor.Process(context.Background(), domain.BirthdayJob{
		ID:          "birthday-2026-05-21-user-123",
		UserID:      "123",
		Date:        time.Date(2026, 5, 21, 7, 0, 0, 0, time.UTC),
		VoucherCode: "BIRTHDAY_123_20260521",
		User: domain.User{
			ID:       "123",
			Name:     "Garvin",
			Email:    "garvin@example.test",
			IsActive: true,
		},
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if issuer.requests != 0 {
		t.Fatalf("expected no voucher issue request, got %d", issuer.requests)
	}
	if sender.messages[0].SubstitutionData["voucher_code"] != "BIRTHDAY_123_20260521" {
		t.Fatalf("expected job voucher code, got %#v", sender.messages[0].SubstitutionData)
	}
}

func TestProcessorRendersHTMLTemplateWhenRendererConfigured(t *testing.T) {
	sender := &fakeSender{}
	processor := NewProcessor(nil, sender, fakeValidator{valid: true}, &fakeVoucherIssuer{code: "HBD-GARVIN"}, campaign.BirthdayCampaign{
		TemplateIDs: []string{"birthday1.html"},
		ActionURL:   "https://kyou.id/user/my-voucher",
		RandomIntn:  func(int) int { return 0 },
	})
	processor.Domain = "kyou.id"
	processor.FromEmail = "nandayo@kyou.id"
	processor.FromName = "Kyou.id"
	processor.Renderer = fakeRenderer{
		subject: "Selamat ulang tahun, Garvin",
		html:    "<h1>Happy birthday Garvin</h1>",
	}

	_, err := processor.Process(context.Background(), domain.BirthdayJob{
		ID:     "birthday-2026-05-21-user-123",
		UserID: "123",
		Date:   time.Date(2026, 5, 21, 7, 0, 0, 0, time.UTC),
		User: domain.User{
			ID:       "123",
			Name:     "Garvin",
			Email:    "garvin@example.test",
			IsActive: true,
		},
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	msg := sender.messages[0]
	if msg.TemplateID != "birthday1.html" {
		t.Fatalf("expected selected template id, got %q", msg.TemplateID)
	}
	if msg.Subject != "Selamat ulang tahun, Garvin" || msg.HTMLBody != "<h1>Happy birthday Garvin</h1>" {
		t.Fatalf("expected rendered email fields, got %#v", msg)
	}
	if msg.TextBody == "" {
		t.Fatalf("expected plain text fallback")
	}
}

type fakeRepository struct {
	user      domain.User
	wishlist  []domain.WishlistItem
	fyp       []domain.FYPItem
	popular   []domain.FYPItem
	converted bool
}

func (r *fakeRepository) BirthdayUsers(context.Context, string) ([]domain.User, error) {
	return []domain.User{r.user}, nil
}

func (r *fakeRepository) UserByID(context.Context, string) (domain.User, error) {
	return r.user, nil
}

func (r *fakeRepository) Wishlist(context.Context, string) ([]domain.WishlistItem, error) {
	return r.wishlist, nil
}

func (r *fakeRepository) FYP(context.Context, string) ([]domain.FYPItem, error) {
	return r.fyp, nil
}

func (r *fakeRepository) Popular(context.Context) ([]domain.FYPItem, error) {
	return r.popular, nil
}

func (r *fakeRepository) HasConverted(context.Context, string, time.Time, time.Time) (bool, error) {
	return r.converted, nil
}

func (r *fakeRepository) SuppressEmail(context.Context, string, string) error {
	return nil
}

type fakeSender struct {
	messages []domain.EmailMessage
}

func (s *fakeSender) SendTemplate(_ context.Context, msg domain.EmailMessage) (domain.SendResult, error) {
	s.messages = append(s.messages, msg)
	return domain.SendResult{MessageID: "msg_1"}, nil
}

type fakeValidator struct {
	valid bool
}

func (v fakeValidator) Validate(context.Context, string) (bool, error) {
	return v.valid, nil
}

type fakeVoucherIssuer struct {
	code     string
	requests int
}

func (i *fakeVoucherIssuer) IssueBirthdayVoucher(context.Context, domain.User, time.Time) (string, error) {
	i.requests++
	return i.code, nil
}

type fakeRenderer struct {
	subject string
	html    string
}

func (r fakeRenderer) Render(_ string, _ map[string]any) (string, string, error) {
	return r.subject, r.html, nil
}
