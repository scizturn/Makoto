package worker

import (
	"context"

	"github.com/kyou-id/makoto/internal/campaign"
	"github.com/kyou-id/makoto/internal/domain"
	"github.com/kyou-id/makoto/internal/email"
	"github.com/kyou-id/makoto/internal/voucher"
)

type AnniversaryProcessor struct {
	store     Store
	sender    email.Sender
	validator email.Validator
	vouchers  voucher.Issuer
	campaign  campaign.AnniversaryCampaign
	Renderer  Renderer

	Domain    string
	FromEmail string
	FromName  string

	BeforeSend func(ctx context.Context, job domain.AnniversaryJob, result ProcessResult) error
}

func NewAnniversaryProcessor(store Store, sender email.Sender, validator email.Validator, vouchers voucher.Issuer, campaign campaign.AnniversaryCampaign) *AnniversaryProcessor {
	return &AnniversaryProcessor{
		store:     store,
		sender:    sender,
		validator: validator,
		vouchers:  vouchers,
		campaign:  campaign,
	}
}

func (p *AnniversaryProcessor) Process(ctx context.Context, job domain.AnniversaryJob) (ProcessResult, error) {
	user := job.User
	if user.ID == "" && p.store != nil {
		var err error
		user, err = p.store.UserByID(ctx, job.UserID)
		if err != nil {
			return ProcessResult{}, err
		}
	}

	if !user.IsActive || user.Email == "" {
		return ProcessResult{}, nil
	}

	valid, err := p.validator.Validate(ctx, user.Email)
	if err != nil {
		return ProcessResult{}, err
	}
	if !valid {
		if p.store != nil {
			return ProcessResult{}, p.store.SuppressEmail(ctx, user.Email, "email validation failed")
		}
		return ProcessResult{}, nil
	}

	voucherCode := job.VoucherCode
	
	wishlist, err := p.resolvePersonalization(ctx, job, user.ID)
	if err != nil {
		return ProcessResult{}, err
	}

	templateID := p.campaign.SelectTemplate(job.Date, job.ID)
	mergeData := p.campaign.BuildMergeData(user, voucherCode, wishlist, job.Years, job.HistoricalItem, job.Date)
	result := ProcessResult{
		TemplateID: templateID,
		ActionURL:  mergeString(mergeData, "action_url"),
	}
	msg := domain.EmailMessage{
		Domain:           p.Domain,
		FromEmail:        p.FromEmail,
		FromName:         p.FromName,
		ToEmail:          user.Email,
		TemplateID:       templateID,
		SubstitutionData: mergeData,
	}
	if p.Renderer != nil {
		subject, html, err := p.Renderer.Render(templateID, mergeData)
		if err != nil {
			return ProcessResult{}, err
		}
		msg.Subject = subject
		msg.HTMLBody = html
		msg.TextBody = subject + "\n\n" + user.Name + ", voucher: " + voucherCode
		result.Subject = subject
	}
	if p.BeforeSend != nil {
		if err := p.BeforeSend(ctx, job, result); err != nil {
			return ProcessResult{}, err
		}
	}
	sendResult, err := p.sender.SendTemplate(ctx, msg)
	result.SendResult = sendResult
	return result, err
}

func (p *AnniversaryProcessor) resolvePersonalization(ctx context.Context, job domain.AnniversaryJob, userID string) ([]domain.WishlistItem, error) {
	if p.store == nil {
		return job.WishlistItems, nil
	}

	wishlist, err := p.store.Wishlist(ctx, userID)
	if err != nil {
		return nil, err
	}
	popular, err := p.store.Popular(ctx)
	if err != nil {
		return nil, err
	}

	for _, pop := range popular {
		if len(wishlist) >= 3 {
			break
		}
		wishlist = append(wishlist, campaign.FYPToWishlistItem(pop))
	}

	return wishlist, nil
}
