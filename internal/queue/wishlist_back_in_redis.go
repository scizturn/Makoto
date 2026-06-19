package queue

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/kyou-id/makoto/internal/domain"
	"github.com/redis/go-redis/v9"
)

type WishlistBackInRedisQueue struct {
	client *redis.Client
	name   string
}

type WishlistBackInProcessingJob struct {
	Job     domain.WishlistBackInJob
	Payload string
}

func NewWishlistBackInRedisQueue(addr, password string, db int, name string) *WishlistBackInRedisQueue {
	if name == "" {
		name = "wishlist_back_in_email_jobs"
	}
	return &WishlistBackInRedisQueue{client: redis.NewClient(&redis.Options{Addr: addr, Password: password, DB: db}), name: name}
}

func (q *WishlistBackInRedisQueue) Close() error { return q.client.Close() }

func (q *WishlistBackInRedisQueue) Dequeue(ctx context.Context, timeout time.Duration) (WishlistBackInProcessingJob, error) {
	payload, err := q.client.BLMove(ctx, q.name, q.processingName(), "LEFT", "RIGHT", timeout).Result()
	if err != nil {
		return WishlistBackInProcessingJob{}, err
	}
	job, err := DecodeWishlistBackInJob(payload)
	if err != nil {
		return WishlistBackInProcessingJob{}, err
	}
	return WishlistBackInProcessingJob{Job: job, Payload: payload}, nil
}

func (q *WishlistBackInRedisQueue) Ack(ctx context.Context, payload string) error {
	removed, err := q.client.LRem(ctx, q.processingName(), 1, payload).Result()
	if err != nil {
		return err
	}
	if removed == 0 {
		return fmt.Errorf("processing payload was not acknowledged")
	}
	return nil
}

func (q *WishlistBackInRedisQueue) RecoverProcessing(ctx context.Context) (int64, error) {
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

func (q *WishlistBackInRedisQueue) processingName() string { return q.name + ":processing" }
