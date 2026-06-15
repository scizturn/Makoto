package queue

import (
	"encoding/json"

	"github.com/kyou-id/makoto/internal/domain"
)

func EncodeLeftoverCartJob(job domain.LeftoverCartJob) (string, error) {
	payload, err := json.Marshal(job)
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

func DecodeLeftoverCartJob(payload string) (domain.LeftoverCartJob, error) {
	var job domain.LeftoverCartJob
	if err := json.Unmarshal([]byte(payload), &job); err != nil {
		return domain.LeftoverCartJob{}, err
	}
	return job, nil
}
