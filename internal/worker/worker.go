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
	Renderer  Renderer

	Domain    string
	FromEmail string
	FromName  string

	BeforeSend func(ctx context.Context, job domain.BirthdayJob, result ProcessResult) error
}

type ProcessResult struct {
	TemplateID string
	Subject    string
	ActionURL  string
	SendResult domain.SendResult
	Outcome    ProcessOutcome
	SkipReason string
}

type ProcessOutcome string

const (
	ProcessOutcomeSent    ProcessOutcome = "sent"
	ProcessOutcomeSkipped ProcessOutcome = "skipped"
)

type Renderer interface {
	Render(templateID string, mergeData map[string]any) (subject string, html string, err error)
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

func (p *Processor) Process(ctx context.Context, job domain.BirthdayJob) (ProcessResult, error) {
	user, err := p.resolveUser(ctx, job)
	if err != nil {
		return ProcessResult{}, err
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

	start := time.Date(job.Date.Year(), job.Date.Month(), job.Date.Day(), 0, 0, 0, 0, job.Date.Location())
	if p.store != nil {
		converted, err := p.store.HasConverted(ctx, user.ID, start, start.Add(14*24*time.Hour))
		if err != nil {
			return ProcessResult{}, err
		}
		if converted {
			log.Printf("birthday email skipped because user converted: user_id=%s", user.ID)
			return ProcessResult{}, nil
		}
	}

	voucherCode := job.VoucherCode
	if voucherCode == "" {
		if p.vouchers == nil {
			return ProcessResult{}, fmt.Errorf("birthday job %q missing voucher_code", job.ID)
		}
		voucherCode, err = p.vouchers.IssueBirthdayVoucher(ctx, user, job.Date)
		if err != nil {
			return ProcessResult{}, err
		}
	}

	wishlist, fyp, popular, err := p.resolvePersonalization(ctx, job, user.ID)
	if err != nil {
		return ProcessResult{}, err
	}

	templateID := p.campaign.SelectTemplate(job.Date, templateSelectionKey(job, user))
	mergeData := p.campaign.BuildMergeData(user, voucherCode, wishlist, fyp, popular)
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
		if subjectTpl := p.campaign.SelectSubject(job.Date, templateSelectionKey(job, user)); subjectTpl != "" {
			subject, err = p.campaign.RenderSubject(subjectTpl, user)
			if err != nil {
				return ProcessResult{}, err
			}
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

func templateSelectionKey(job domain.BirthdayJob, user domain.User) string {
	if job.ID != "" {
		return job.ID
	}
	if user.ID != "" {
		return user.ID
	}
	return job.UserID
}

func mergeString(values map[string]any, key string) string {
	value, _ := values[key].(string)
	return value
}
