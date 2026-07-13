package email

import (
	"context"
	"log"
	"net/mail"

	"github.com/kyou-id/makoto/internal/domain"
	"github.com/kyou-id/makoto/internal/mask"
)

type Sender interface {
	SendTemplate(ctx context.Context, msg domain.EmailMessage) (domain.SendResult, error)
}

type Validator interface {
	Validate(ctx context.Context, email string) (bool, error)
}

type LoggingSender struct{}

func (LoggingSender) SendTemplate(_ context.Context, msg domain.EmailMessage) (domain.SendResult, error) {
	log.Printf("template email queued locally: to=%s template=%s", mask.Email(msg.ToEmail), msg.TemplateID)
	return domain.SendResult{MessageID: "local-message"}, nil
}

type AllowAllValidator struct{}

func (AllowAllValidator) Validate(context.Context, string) (bool, error) {
	return true, nil
}

// FailOpenValidator lets an address through when the validation provider is
// unreachable — an outage on their side must not cost us a campaign.
//
// It does NOT fail open on a syntactically broken address. That is not an outage,
// and letting it through only moves the rejection downstream, where Kirim.email
// answers 422 "The to.0 field must be a valid email address" and the job retries
// until it dead-letters. Their validation API returns 500 (not "invalid") on such
// addresses, so without this check the two failures conspire: the validator cannot
// judge it, we fail open, and the send fails anyway.
type FailOpenValidator struct {
	Validator Validator
}

func (v FailOpenValidator) Validate(ctx context.Context, address string) (bool, error) {
	if _, err := mail.ParseAddress(address); err != nil {
		log.Printf("email rejected as malformed: email=%s err=%v", mask.Email(address), err)
		return false, nil
	}

	valid, err := v.Validator.Validate(ctx, address)
	if err != nil {
		log.Printf("email validation failed open: email=%s err=%v", mask.Email(address), err)
		return true, nil
	}
	return valid, nil
}
