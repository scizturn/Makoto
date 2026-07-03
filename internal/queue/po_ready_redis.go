package queue

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/kyou-id/makoto/internal/domain"
	"github.com/redis/go-redis/v9"
)

type PoReadyRedisQueue struct {
	client *redis.Client
	name   string
}

type PoReadyProcessingJob struct {
	Job     domain.PoReadyJob
	Payload string
}

func NewPoReadyRedisQueue(addr, password string, db int, name string) *PoReadyRedisQueue {
	if name == "" {
		name = "po_ready_email_jobs"
	}
	return &PoReadyRedisQueue{
		client: redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: password,
			DB:       db,
		}),
		name: name,
	}
}

func (q *PoReadyRedisQueue) Close() error {
	return q.client.Close()
}

func (q *PoReadyRedisQueue) Enqueue(ctx context.Context, job domain.PoReadyJob) error {
	return q.EnqueueTo(ctx, q.name, job)
}

func (q *PoReadyRedisQueue) EnqueueTo(ctx context.Context, name string, job domain.PoReadyJob) error {
	payload, err := EncodePoReadyJob(job)
	if err != nil {
		return err
	}
	if name == "" {
		name = q.name
	}
	return q.client.RPush(ctx, name, payload).Err()
}

func (q *PoReadyRedisQueue) Dequeue(ctx context.Context, timeout time.Duration) (PoReadyProcessingJob, error) {
	payload, err := q.client.BLMove(ctx, q.name, q.processingName(), "LEFT", "RIGHT", timeout).Result()
	if err != nil {
		return PoReadyProcessingJob{}, err
	}
	job, err := DecodePoReadyJob(payload)
	if err != nil {
		return PoReadyProcessingJob{}, err
	}
	return PoReadyProcessingJob{Job: job, Payload: payload}, nil
}

func (q *PoReadyRedisQueue) Ack(ctx context.Context, payload string) error {
	removed, err := q.client.LRem(ctx, q.processingName(), 1, payload).Result()
	if err != nil {
		return err
	}
	if removed == 0 {
		return fmt.Errorf("processing payload was not acknowledged")
	}
	return nil
}

func (q *PoReadyRedisQueue) RecoverProcessing(ctx context.Context) (int64, error) {
	var recovered int64
	for {
		payload, err := q.client.LMove(ctx, q.processingName(), q.name, "RIGHT", "LEFT").Result()
		if errors.Is(err, redis.Nil) {
			return recovered, nil
		}
		if err != nil {
			return recovered, err
		}
		if payload != "" {
			recovered++
		}
	}
}

func (q *PoReadyRedisQueue) processingName() string {
	return q.name + ":processing"
}
