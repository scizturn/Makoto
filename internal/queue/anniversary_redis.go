package queue

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/kyou-id/makoto/internal/domain"
	"github.com/redis/go-redis/v9"
)

type AnniversaryRedisQueue struct {
	client *redis.Client
	name   string
}

type AnniversaryProcessingJob struct {
	Job     domain.AnniversaryJob
	Payload string
}

func NewAnniversaryRedisQueue(addr, password string, db int, name string) *AnniversaryRedisQueue {
	if name == "" {
		name = "anniversary_email_jobs"
	}
	return &AnniversaryRedisQueue{
		client: redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: password,
			DB:       db,
		}),
		name: name,
	}
}

func (q *AnniversaryRedisQueue) Close() error {
	return q.client.Close()
}

func (q *AnniversaryRedisQueue) Enqueue(ctx context.Context, job domain.AnniversaryJob) error {
	return q.EnqueueTo(ctx, q.name, job)
}

func (q *AnniversaryRedisQueue) EnqueueTo(ctx context.Context, name string, job domain.AnniversaryJob) error {
	payload, err := EncodeAnniversaryJob(job)
	if err != nil {
		return err
	}
	return q.client.RPush(ctx, anniversaryQueueName(name), payload).Err()
}

func (q *AnniversaryRedisQueue) Dequeue(ctx context.Context, timeout time.Duration) (AnniversaryProcessingJob, error) {
	payload, err := q.client.BLMove(ctx, q.name, q.processingName(), "LEFT", "RIGHT", timeout).Result()
	if err != nil {
		return AnniversaryProcessingJob{}, err
	}
	job, err := DecodeAnniversaryJob(payload)
	if err != nil {
		return AnniversaryProcessingJob{}, err
	}
	return AnniversaryProcessingJob{Job: job, Payload: payload}, nil
}

func (q *AnniversaryRedisQueue) Ack(ctx context.Context, payload string) error {
	removed, err := q.client.LRem(ctx, q.processingName(), 1, payload).Result()
	if err != nil {
		return err
	}
	if removed == 0 {
		return fmt.Errorf("processing payload was not acknowledged")
	}
	return nil
}

func (q *AnniversaryRedisQueue) RecoverProcessing(ctx context.Context) (int64, error) {
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

func anniversaryQueueName(name string) string {
	if name == "" {
		return "anniversary_email_jobs"
	}
	return name
}

func (q *AnniversaryRedisQueue) processingName() string {
	return anniversaryQueueName(q.name) + ":processing"
}
