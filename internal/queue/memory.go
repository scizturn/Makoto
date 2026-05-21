package queue

import (
	"context"

	"github.com/kyou-id/makoto/internal/domain"
)

type MemoryQueue struct {
	jobs chan domain.BirthdayJob
}

func NewMemoryQueue(size int) *MemoryQueue {
	if size <= 0 {
		size = 100
	}
	return &MemoryQueue{jobs: make(chan domain.BirthdayJob, size)}
}

func (q *MemoryQueue) Enqueue(ctx context.Context, job domain.BirthdayJob) error {
	select {
	case q.jobs <- job:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (q *MemoryQueue) Jobs() <-chan domain.BirthdayJob {
	return q.jobs
}

func (q *MemoryQueue) Close() {
	close(q.jobs)
}
