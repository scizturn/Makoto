package queue

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/kyou-id/makoto/internal/domain"
	"github.com/redis/go-redis/v9"
)

type LeftoverCartRedisQueue struct {
	client *redis.Client
	name   string
}

type LeftoverCartProcessingJob struct {
	Job     domain.LeftoverCartJob
	Payload string
}

func NewLeftoverCartRedisQueue(addr, password string, db int, name string) *LeftoverCartRedisQueue {
	if name == "" {
		name = "leftover_cart_email_jobs"
	}
	return &LeftoverCartRedisQueue{
		client: redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: password,
			DB:       db,
		}),
		name: name,
	}
}

func (q *LeftoverCartRedisQueue) Close() error {
	return q.client.Close()
}

func (q *LeftoverCartRedisQueue) Dequeue(ctx context.Context, timeout time.Duration) (LeftoverCartProcessingJob, error) {
	payload, err := q.client.BLMove(ctx, q.name, q.processingName(), "LEFT", "RIGHT", timeout).Result()
	if err != nil {
		return LeftoverCartProcessingJob{}, err
	}
	job, err := DecodeLeftoverCartJob(payload)
	if err != nil {
		return LeftoverCartProcessingJob{}, err
	}
	return LeftoverCartProcessingJob{Job: job, Payload: payload}, nil
}

func (q *LeftoverCartRedisQueue) Ack(ctx context.Context, payload string) error {
	removed, err := q.client.LRem(ctx, q.processingName(), 1, payload).Result()
	if err != nil {
		return err
	}
	if removed == 0 {
		return fmt.Errorf("processing payload was not acknowledged")
	}
	return nil
}

func (q *LeftoverCartRedisQueue) RecoverProcessing(ctx context.Context) (int64, error) {
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

func (q *LeftoverCartRedisQueue) processingName() string {
	return q.name + ":processing"
}
