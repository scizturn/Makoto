package worker

import (
	"context"

	"github.com/kyou-id/makoto/internal/campaign"
	"github.com/kyou-id/makoto/internal/domain"
	"github.com/kyou-id/makoto/internal/email"
)

type LeftoverCartProcessor struct {
	sender    email.Sender
	validator email.Validator
	campaign  campaign.LeftoverCartCampaign
	Renderer  Renderer

	Domain    string
	FromEmail string
	FromName  string

	BeforeSend func(ctx context.Context, job domain.LeftoverCartJob, result ProcessResult) error
}

func NewLeftoverCartProcessor(sender email.Sender, validator email.Validator, c campaign.LeftoverCartCampaign) *LeftoverCartProcessor {
	return &LeftoverCartProcessor{
		sender:    sender,
		validator: validator,
		campaign:  c,
	}
}

func (p *LeftoverCartProcessor) Process(ctx context.Context, job domain.LeftoverCartJob) (ProcessResult, error) {
	user := job.User

	if !user.IsActive || user.Email == "" {
		return ProcessResult{}, nil
	}

	valid, err := p.validator.Validate(ctx, user.Email)
	if err != nil {
		return ProcessResult{}, err
	}
	if !valid {
		return ProcessResult{}, nil
	}

	greetingTpl := p.campaign.SelectGreeting(job.Date, job.ID)
	greeting := p.campaign.RenderGreeting(greetingTpl, user)
	templateID := p.campaign.SelectTemplate(job.Date, job.ID)
	mergeData := p.campaign.BuildMergeData(user, job.CartItems, job.RecoItems, job.HistoricalItem, greeting)

	result := ProcessResult{
		TemplateID: templateID,
		ActionURL:  mergeString(mergeData, "cart_url"),
	}
	msg := domain.EmailMessage{
		Domain:           p.Domain,
		FromEmail:        p.FromEmail,
		FromName:         p.FromName,
		ToEmail:          user.Email,
		TemplateID:       templateID,
		SubstitutionData: mergeData,
		Tags:             job.ID,
	}
	if p.Renderer != nil {
		subject, html, err := p.Renderer.Render(templateID, mergeData)
		if err != nil {
			return ProcessResult{}, err
		}
		if subjectTpl := p.campaign.SelectSubject(job.Date, job.ID); subjectTpl != "" {
			subject = p.campaign.RenderSubject(subjectTpl, user)
		}
		msg.Subject = subject
		msg.HTMLBody = html
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
	return result, err
}
