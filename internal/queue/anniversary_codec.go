package queue

import (
	"encoding/json"

	"github.com/kyou-id/makoto/internal/domain"
)

func EncodeAnniversaryJob(job domain.AnniversaryJob) (string, error) {
	payload, err := json.Marshal(job)
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

func DecodeAnniversaryJob(payload string) (domain.AnniversaryJob, error) {
	var job domain.AnniversaryJob
	if err := json.Unmarshal([]byte(payload), &job); err != nil {
		return domain.AnniversaryJob{}, err
	}
	return job, nil
}
