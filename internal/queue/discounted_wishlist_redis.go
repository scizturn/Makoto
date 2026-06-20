package queue

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/kyou-id/makoto/internal/domain"
	"github.com/redis/go-redis/v9"
)

type DiscountedWishlistRedisQueue struct {
	client *redis.Client
	name   string
}

type DiscountedWishlistProcessingJob struct {
	Job     domain.DiscountedWishlistJob
	Payload string
}

func NewDiscountedWishlistRedisQueue(addr, password string, db int, name string) *DiscountedWishlistRedisQueue {
	if name == "" {
		name = "discounted_wishlist_email_jobs"
	}
	return &DiscountedWishlistRedisQueue{
		client: redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: password,
			DB:       db,
		}),
		name: name,
	}
}

func (q *DiscountedWishlistRedisQueue) Close() error {
	return q.client.Close()
}

func (q *DiscountedWishlistRedisQueue) Enqueue(ctx context.Context, job domain.DiscountedWishlistJob) error {
	return q.EnqueueTo(ctx, q.name, job)
}

func (q *DiscountedWishlistRedisQueue) EnqueueTo(ctx context.Context, name string, job domain.DiscountedWishlistJob) error {
	payload, err := EncodeDiscountedWishlistJob(job)
	if err != nil {
		return err
	}
	if name == "" {
		name = q.name
	}
	return q.client.RPush(ctx, name, payload).Err()
}

func (q *DiscountedWishlistRedisQueue) Dequeue(ctx context.Context, timeout time.Duration) (DiscountedWishlistProcessingJob, error) {
	payload, err := q.client.BLMove(ctx, q.name, q.processingName(), "LEFT", "RIGHT", timeout).Result()
	if err != nil {
		return DiscountedWishlistProcessingJob{}, err
	}
	job, err := DecodeDiscountedWishlistJob(payload)
	if err != nil {
		return DiscountedWishlistProcessingJob{}, err
	}
	return DiscountedWishlistProcessingJob{Job: job, Payload: payload}, nil
}

func (q *DiscountedWishlistRedisQueue) Ack(ctx context.Context, payload string) error {
	removed, err := q.client.LRem(ctx, q.processingName(), 1, payload).Result()
	if err != nil {
		return err
	}
	if removed == 0 {
		return fmt.Errorf("processing payload was not acknowledged")
	}
	return nil
}

func (q *DiscountedWishlistRedisQueue) RecoverProcessing(ctx context.Context) (int64, error) {
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

func (q *DiscountedWishlistRedisQueue) processingName() string {
	return q.name + ":processing"
}
