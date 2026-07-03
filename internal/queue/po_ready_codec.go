package queue

import (
	"encoding/json"

	"github.com/kyou-id/makoto/internal/domain"
)

func EncodePoReadyJob(job domain.PoReadyJob) (string, error) {
	payload, err := json.Marshal(job)
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

func DecodePoReadyJob(payload string) (domain.PoReadyJob, error) {
	var job domain.PoReadyJob
	if err := json.Unmarshal([]byte(payload), &job); err != nil {
		return domain.PoReadyJob{}, err
	}
	return job, nil
}
