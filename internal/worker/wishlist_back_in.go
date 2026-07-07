package worker

import (
	"context"

	"github.com/kyou-id/makoto/internal/campaign"
	"github.com/kyou-id/makoto/internal/domain"
	"github.com/kyou-id/makoto/internal/email"
)

type WishlistBackInProcessor struct {
	sender    email.Sender
	validator email.Validator
	campaign  campaign.WishlistBackInCampaign
	Renderer  Renderer

	Domain    string
	FromEmail string
	FromName  string

	BeforeSend func(context.Context, domain.WishlistBackInJob, ProcessResult) error
}

func NewWishlistBackInProcessor(sender email.Sender, validator email.Validator, c campaign.WishlistBackInCampaign) *WishlistBackInProcessor {
	return &WishlistBackInProcessor{sender: sender, validator: validator, campaign: c}
}

func (p *WishlistBackInProcessor) Process(ctx context.Context, job domain.WishlistBackInJob) (ProcessResult, error) {
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

	greeting := p.campaign.RenderGreeting(p.campaign.SelectGreeting(job.Date, job.ID), user)
	templateID := p.campaign.SelectTemplate(job.Date, job.ID)
	mergeData := p.campaign.BuildMergeData(user, job.VoucherCode, job.Items, job.CompanionItem, greeting)
	result := ProcessResult{TemplateID: templateID, ActionURL: mergeString(mergeData, "action_url")}
	message := domain.EmailMessage{
		Domain: p.Domain, FromEmail: p.FromEmail, FromName: p.FromName,
		ToEmail: user.Email, TemplateID: templateID, SubstitutionData: mergeData,
	}
	if p.Renderer != nil {
		subject, body, err := p.Renderer.Render(templateID, mergeData)
		if err != nil {
			return ProcessResult{}, err
		}
		message.Subject = subject
		message.HTMLBody = body
		message.TextBody = subject + "\n\n" + greeting
		result.Subject = subject
	}
	if p.BeforeSend != nil {
		if err := p.BeforeSend(ctx, job, result); err != nil {
			return ProcessResult{}, err
		}
	}
	sendResult, err := p.sender.SendTemplate(ctx, message)
	result.SendResult = sendResult
	return result, err
}
