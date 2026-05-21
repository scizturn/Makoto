package worker

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/kyou-id/makoto/internal/campaign"
	"github.com/kyou-id/makoto/internal/domain"
	"github.com/kyou-id/makoto/internal/email"
	"github.com/kyou-id/makoto/internal/voucher"
)

type Store interface {
	UserByID(ctx context.Context, userID string) (domain.User, error)
	Wishlist(ctx context.Context, userID string) ([]domain.WishlistItem, error)
	FYP(ctx context.Context, userID string) ([]domain.FYPItem, error)
	Popular(ctx context.Context) ([]domain.FYPItem, error)
	HasConverted(ctx context.Context, userID string, from time.Time, to time.Time) (bool, error)
	SuppressEmail(ctx context.Context, email string, reason string) error
}

type Processor struct {
	store     Store
	sender    email.Sender
	validator email.Validator
	vouchers  voucher.Issuer
	campaign  campaign.BirthdayCampaign

	Domain    string
	FromEmail string
	FromName  string
}

func NewProcessor(store Store, sender email.Sender, validator email.Validator, vouchers voucher.Issuer, campaign campaign.BirthdayCampaign) *Processor {
	return &Processor{
		store:     store,
		sender:    sender,
		validator: validator,
		vouchers:  vouchers,
		campaign:  campaign,
	}
}

func (p *Processor) Process(ctx context.Context, job domain.BirthdayJob) error {
	user, err := p.resolveUser(ctx, job)
	if err != nil {
		return err
	}

	if !user.IsActive || user.Email == "" {
		return nil
	}

	valid, err := p.validator.Validate(ctx, user.Email)
	if err != nil {
		return err
	}
	if !valid {
		if p.store != nil {
			return p.store.SuppressEmail(ctx, user.Email, "email validation failed")
		}
		return nil
	}

	start := time.Date(job.Date.Year(), job.Date.Month(), job.Date.Day(), 0, 0, 0, 0, job.Date.Location())
	if p.store != nil {
		converted, err := p.store.HasConverted(ctx, user.ID, start, start.Add(14*24*time.Hour))
		if err != nil {
			return err
		}
		if converted {
			log.Printf("birthday email skipped because user converted: user_id=%s", user.ID)
			return nil
		}
	}

	voucherCode, err := p.vouchers.IssueBirthdayVoucher(ctx, user, job.Date)
	if err != nil {
		return err
	}

	wishlist, fyp, popular, err := p.resolvePersonalization(ctx, job, user.ID)
	if err != nil {
		return err
	}

	msg := domain.EmailMessage{
		Domain:           p.Domain,
		FromEmail:        p.FromEmail,
		FromName:         p.FromName,
		ToEmail:          user.Email,
		TemplateID:       p.campaign.SelectTemplate(job.Date),
		SubstitutionData: p.campaign.BuildMergeData(user, voucherCode, wishlist, fyp, popular),
	}
	_, err = p.sender.SendTemplate(ctx, msg)
	return err
}

func (p *Processor) resolveUser(ctx context.Context, job domain.BirthdayJob) (domain.User, error) {
	if job.User.ID != "" || job.User.Email != "" {
		return job.User, nil
	}
	if p.store == nil {
		return domain.User{}, fmt.Errorf("birthday job %q missing user payload", job.ID)
	}
	return p.store.UserByID(ctx, job.UserID)
}

func (p *Processor) resolvePersonalization(ctx context.Context, job domain.BirthdayJob, userID string) ([]domain.WishlistItem, []domain.FYPItem, []domain.FYPItem, error) {
	if p.store == nil {
		return job.WishlistItems, job.FYPItems, job.PopularItems, nil
	}

	wishlist, err := p.store.Wishlist(ctx, userID)
	if err != nil {
		return nil, nil, nil, err
	}
	fyp, err := p.store.FYP(ctx, userID)
	if err != nil {
		return nil, nil, nil, err
	}
	popular, err := p.store.Popular(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	return wishlist, fyp, popular, nil
}
