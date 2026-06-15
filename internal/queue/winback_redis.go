package queue

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/kyou-id/makoto/internal/domain"
	"github.com/redis/go-redis/v9"
)

type WinbackRedisQueue struct {
	client *redis.Client
	name   string
}

type WinbackProcessingJob struct {
	Job     domain.WinbackJob
	Payload string
}

func NewWinbackRedisQueue(addr, password string, db int, name string) *WinbackRedisQueue {
	if name == "" {
		name = "winback_email_jobs"
	}
	return &WinbackRedisQueue{
		client: redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: password,
			DB:       db,
		}),
		name: name,
	}
}

func (q *WinbackRedisQueue) Close() error {
	return q.client.Close()
}

func (q *WinbackRedisQueue) Dequeue(ctx context.Context, timeout time.Duration) (WinbackProcessingJob, error) {
	payload, err := q.client.BLMove(ctx, q.name, q.processingName(), "LEFT", "RIGHT", timeout).Result()
	if err != nil {
		return WinbackProcessingJob{}, err
	}
	job, err := DecodeWinbackJob(payload)
	if err != nil {
		return WinbackProcessingJob{}, err
	}
	return WinbackProcessingJob{Job: job, Payload: payload}, nil
}

func (q *WinbackRedisQueue) Ack(ctx context.Context, payload string) error {
	removed, err := q.client.LRem(ctx, q.processingName(), 1, payload).Result()
	if err != nil {
		return err
	}
	if removed == 0 {
		return fmt.Errorf("processing payload was not acknowledged")
	}
	return nil
}

func (q *WinbackRedisQueue) RecoverProcessing(ctx context.Context) (int64, error) {
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

func (q *WinbackRedisQueue) processingName() string {
	return q.name + ":processing"
}
