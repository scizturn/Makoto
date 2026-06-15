package queue

import (
	"encoding/json"

	"github.com/kyou-id/makoto/internal/domain"
)

func EncodeDiscountedWishlistJob(job domain.DiscountedWishlistJob) (string, error) {
	payload, err := json.Marshal(job)
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

func DecodeDiscountedWishlistJob(payload string) (domain.DiscountedWishlistJob, error) {
	var job domain.DiscountedWishlistJob
	if err := json.Unmarshal([]byte(payload), &job); err != nil {
		return domain.DiscountedWishlistJob{}, err
	}
	return job, nil
}
