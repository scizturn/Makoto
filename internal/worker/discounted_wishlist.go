package worker

import (
	"context"

	"github.com/kyou-id/makoto/internal/campaign"
	"github.com/kyou-id/makoto/internal/domain"
	"github.com/kyou-id/makoto/internal/email"
)

type DiscountedWishlistProcessor struct {
	sender    email.Sender
	validator email.Validator
	campaign  campaign.DiscountedWishlistCampaign
	Renderer  Renderer

	Domain    string
	FromEmail string
	FromName  string

	BeforeSend func(ctx context.Context, job domain.DiscountedWishlistJob, result ProcessResult) error
}

func NewDiscountedWishlistProcessor(sender email.Sender, validator email.Validator, c campaign.DiscountedWishlistCampaign) *DiscountedWishlistProcessor {
	return &DiscountedWishlistProcessor{
		sender:    sender,
		validator: validator,
		campaign:  c,
	}
}

func (p *DiscountedWishlistProcessor) Process(ctx context.Context, job domain.DiscountedWishlistJob) (ProcessResult, error) {
	user := job.User

	if !user.IsActive {
		return ProcessResult{Outcome: ProcessOutcomeSkipped, SkipReason: "inactive_user"}, nil
	}
	if user.Email == "" {
		return ProcessResult{Outcome: ProcessOutcomeSkipped, SkipReason: "missing_email"}, nil
	}

	valid, err := p.validator.Validate(ctx, user.Email)
	if err != nil {
		return ProcessResult{}, err
	}
	if !valid {
		return ProcessResult{Outcome: ProcessOutcomeSkipped, SkipReason: "email_validation_rejected"}, nil
	}

	greetingTpl := p.campaign.SelectGreeting(job.Date, job.ID)
	greeting := p.campaign.RenderGreeting(greetingTpl, user)
	templateID := p.campaign.SelectTemplate(job.Date, job.ID)
	mergeData := p.campaign.BuildMergeData(user, job.Items, greeting)

	result := ProcessResult{
		TemplateID: templateID,
		ActionURL:  mergeString(mergeData, "wishlist_url"),
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
		subject, htmlBody, err := p.Renderer.Render(templateID, mergeData)
		if err != nil {
			return ProcessResult{}, err
		}
		if subjectTpl := p.campaign.SelectSubject(job.Date, job.ID); subjectTpl != "" {
			subject, err = p.campaign.RenderSubject(subjectTpl, user)
			if err != nil {
				return ProcessResult{}, err
			}
		}
		msg.Subject = subject
		msg.HTMLBody = htmlBody
		msg.TextBody = subject + "\n\n" + user.Name
		result.Subject = subject
	}
	if p.BeforeSend != nil {
		if err := p.BeforeSend(ctx, job, result); err != nil {
			return ProcessResult{}, err
		}
	}
	sendResult, err := p.sender.SendTemplate(ctx, msg)
	result.SendResult = sendResult
	if err == nil {
		result.Outcome = ProcessOutcomeSent
	}
	return result, err
}
