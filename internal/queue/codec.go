package queue

import (
	"encoding/json"

	"github.com/kyou-id/makoto/internal/domain"
)

func EncodeBirthdayJob(job domain.BirthdayJob) (string, error) {
	payload, err := json.Marshal(job)
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

func DecodeBirthdayJob(payload string) (domain.BirthdayJob, error) {
	var job domain.BirthdayJob
	if err := json.Unmarshal([]byte(payload), &job); err != nil {
		return domain.BirthdayJob{}, err
	}
	return job, nil
}
