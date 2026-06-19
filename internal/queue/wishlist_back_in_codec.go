package queue

import (
	"encoding/json"

	"github.com/kyou-id/makoto/internal/domain"
)

func EncodeWishlistBackInJob(job domain.WishlistBackInJob) (string, error) {
	payload, err := json.Marshal(job)
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

func DecodeWishlistBackInJob(payload string) (domain.WishlistBackInJob, error) {
	var job domain.WishlistBackInJob
	if err := json.Unmarshal([]byte(payload), &job); err != nil {
		return domain.WishlistBackInJob{}, err
	}
	return job, nil
}
