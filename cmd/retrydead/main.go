package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/kyou-id/makoto/internal/config"
	"github.com/kyou-id/makoto/internal/queue"
)

func main() {
	jobID := flag.String("job-id", "", "dead-letter job id to retry")
	flag.Parse()
	if *jobID == "" {
		log.Fatal("missing required -job-id")
	}

	ctx := context.Background()
	cfg := config.Load()
	redisQueue := queue.NewRedisQueue(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB, cfg.QueueName)
	defer func() {
		if err := redisQueue.Close(); err != nil {
			log.Printf("redis close failed: %v", err)
		}
	}()

	jobs, payloads, err := redisQueue.Jobs(ctx, cfg.DeadLetterQueue)
	if err != nil {
		log.Fatalf("read dead-letter queue: %v", err)
	}

	for index, job := range jobs {
		if job.ID != *jobID {
			continue
		}
		if err := redisQueue.RemovePayload(ctx, cfg.DeadLetterQueue, payloads[index]); err != nil {
			log.Fatalf("remove dead-letter payload: %v", err)
		}
		job.Attempt = 1
		if err := redisQueue.Enqueue(ctx, job); err != nil {
			log.Fatalf("requeue job: %v", err)
		}
		fmt.Printf("retried job_id=%s user_id=%s queue=%s\n", job.ID, job.UserID, cfg.QueueName)
		return
	}

	log.Fatalf("job_id %q not found in %s", *jobID, cfg.DeadLetterQueue)
}
