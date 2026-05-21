package voucher

import (
	"context"
	"time"

	"github.com/kyou-id/makoto/internal/domain"
)

type Issuer interface {
	IssueBirthdayVoucher(ctx context.Context, user domain.User, birthdayDate time.Time) (string, error)
}

type StaticIssuer struct {
	Code string
}

func (i StaticIssuer) IssueBirthdayVoucher(context.Context, domain.User, time.Time) (string, error) {
	return i.Code, nil
}
