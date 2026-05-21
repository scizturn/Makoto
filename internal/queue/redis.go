package queue

import (
	"context"
	"fmt"
	"time"

	"github.com/kyou-id/makoto/internal/domain"
	"github.com/redis/go-redis/v9"
)

type RedisQueue struct {
	client *redis.Client
	name   string
}

func NewRedisQueue(addr, password string, db int, name string) *RedisQueue {
	if name == "" {
		name = "birthday_email_jobs"
	}
	return &RedisQueue{
		client: redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: password,
			DB:       db,
		}),
		name: name,
	}
}

func (q *RedisQueue) Close() error {
	return q.client.Close()
}

func (q *RedisQueue) Enqueue(ctx context.Context, job domain.BirthdayJob) error {
	payload, err := EncodeBirthdayJob(job)
	if err != nil {
		return err
	}
	return q.client.LPush(ctx, q.name, payload).Err()
}

func (q *RedisQueue) Dequeue(ctx context.Context, timeout time.Duration) (domain.BirthdayJob, error) {
	result, err := q.client.BRPop(ctx, timeout, q.name).Result()
	if err != nil {
		return domain.BirthdayJob{}, err
	}
	if len(result) != 2 {
		return domain.BirthdayJob{}, fmt.Errorf("unexpected redis pop result: %#v", result)
	}
	return DecodeBirthdayJob(result[1])
}
