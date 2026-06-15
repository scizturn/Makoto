package queue

import (
	"encoding/json"

	"github.com/kyou-id/makoto/internal/domain"
)

func EncodeWinbackJob(job domain.WinbackJob) (string, error) {
	payload, err := json.Marshal(job)
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

func DecodeWinbackJob(payload string) (domain.WinbackJob, error) {
	var job domain.WinbackJob
	if err := json.Unmarshal([]byte(payload), &job); err != nil {
		return domain.WinbackJob{}, err
	}
	return job, nil
}
