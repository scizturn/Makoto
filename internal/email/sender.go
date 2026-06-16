package email

import (
	"context"
	"log"

	"github.com/kyou-id/makoto/internal/domain"
)

type Sender interface {
	SendTemplate(ctx context.Context, msg domain.EmailMessage) (domain.SendResult, error)
}

type Validator interface {
	Validate(ctx context.Context, email string) (bool, error)
}

type LoggingSender struct{}

func (LoggingSender) SendTemplate(_ context.Context, msg domain.EmailMessage) (domain.SendResult, error) {
	log.Printf("template email queued locally: to=%s template=%s", msg.ToEmail, msg.TemplateID)
	return domain.SendResult{MessageID: "local-message"}, nil
}

type AllowAllValidator struct{}

func (AllowAllValidator) Validate(context.Context, string) (bool, error) {
	return true, nil
}

type FailOpenValidator struct {
	Validator Validator
}

func (v FailOpenValidator) Validate(ctx context.Context, address string) (bool, error) {
	valid, err := v.Validator.Validate(ctx, address)
	if err != nil {
		log.Printf("email validation failed open: email=%s err=%v", address, err)
		return true, nil
	}
	return valid, nil
}
