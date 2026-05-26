package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/kyou-id/makoto/internal/campaign"
	"github.com/kyou-id/makoto/internal/config"
	"github.com/kyou-id/makoto/internal/domain"
	"github.com/kyou-id/makoto/internal/email"
	"github.com/kyou-id/makoto/internal/emailtemplate"
	"github.com/kyou-id/makoto/internal/notify"
	"github.com/kyou-id/makoto/internal/queue"
	"github.com/kyou-id/makoto/internal/voucher"
	"github.com/kyou-id/makoto/internal/worker"
	"github.com/redis/go-redis/v9"
)

func main() {
	ctx := context.Background()
	cfg := config.Load()

	birthdayCampaign := campaign.BirthdayCampaign{
		TemplateIDs: cfg.TemplateIDs,
		Closing:     "Selamat merayakan hari spesialmu di Kyou!",
		ActionURL:   cfg.ActionURL,
	}
	sender, validator := buildEmail(cfg)
	voucherIssuer := buildVoucherIssuer(cfg)
	processor := worker.NewProcessor(nil, sender, validator, voucherIssuer, birthdayCampaign)
	processor.Domain = cfg.KirimEmailDomain
	processor.FromEmail = cfg.FromEmail
	processor.FromName = cfg.FromName
	if cfg.EmailTemplateDir != "" {
		processor.Renderer = emailtemplate.FileRenderer{
			Dir:     cfg.EmailTemplateDir,
			Subject: cfg.EmailSubject,
		}
	}

	redisQueue := queue.NewRedisQueue(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB, cfg.QueueName)
	defer func() {
		if err := redisQueue.Close(); err != nil {
			log.Printf("redis close failed: %v", err)
		}
	}()

	discord := notify.DiscordLogger{
		WebhookURL: cfg.DiscordWebhookURL,
		Enabled:    cfg.DiscordEnabled,
	}

	runSender(ctx, redisQueue, processor, discord, senderConfig{
		rateLimitPerMinute: cfg.RateLimitPerMinute,
		deadLetterQueue:    cfg.DeadLetterQueue,
		maxAttempts:        cfg.MaxAttempts,
	})
}

func buildVoucherIssuer(cfg config.Config) voucher.Issuer {
	if cfg.KyouIDAPIToken == "" {
		log.Print("KYOU_ID_API_TOKEN is empty; using local static voucher issuer")
		return voucher.StaticIssuer{Code: "LOCAL-BIRTHDAY"}
	}

	return voucher.KyouClient{
		BaseURL: cfg.KyouIDAPIBaseURL,
		Token:   cfg.KyouIDAPIToken,
	}
}

func buildEmail(cfg config.Config) (email.Sender, email.Validator) {
	if cfg.KirimEmailUsername == "" || cfg.KirimEmailAPIToken == "" {
		log.Print("Kirim.email credentials are empty; using local logging sender and allow-all validator")
		return email.LoggingSender{}, email.AllowAllValidator{}
	}

	client := email.KirimClient{
		BaseURL:  cfg.KirimEmailBaseURL,
		Username: cfg.KirimEmailUsername,
		APIToken: cfg.KirimEmailAPIToken,
	}
	if !cfg.KirimEmailValidate {
		log.Print("KIRIM_EMAIL_VALIDATE is false; skipping strict email validation")
		return client, email.AllowAllValidator{}
	}
	return client, client
}

type senderConfig struct {
	rateLimitPerMinute int
	deadLetterQueue    string
	maxAttempts        int
}

func runSender(ctx context.Context, redisQueue *queue.RedisQueue, processor *worker.Processor, discord notify.DiscordLogger, cfg senderConfig) {
	if cfg.rateLimitPerMinute <= 0 {
		cfg.rateLimitPerMinute = 100
	}
	if cfg.maxAttempts <= 0 {
		cfg.maxAttempts = 3
	}
	interval := time.Minute / time.Duration(cfg.rateLimitPerMinute)
	if interval <= 0 {
		interval = time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		job, err := redisQueue.Dequeue(ctx, 5*time.Second)
		if errors.Is(err, redis.Nil) {
			continue
		}
		if err != nil {
			log.Printf("redis dequeue failed: %v", err)
			continue
		}

		select {
		case <-ctx.Done():
			log.Print(ctx.Err())
			return
		case <-ticker.C:
		}

		if err := processor.Process(ctx, job); err != nil {
			result, failureErr := handleFailedJob(ctx, redisQueue, job, cfg)
			if failureErr != nil {
				log.Printf("birthday email failure handling failed: job_id=%s user_id=%s err=%v failure_err=%v", job.ID, job.UserID, err, failureErr)
				_ = discord.Log(ctx, fmt.Sprintf("[Birthday Email Failure Handling Failed]\nJob: %s\nUser: %s\nEmail: %s\nError: %v\nFailure handling error: %v", job.ID, job.UserID, maskEmail(job.User.Email), err, failureErr))
				continue
			}
			log.Printf("birthday email failed: job_id=%s user_id=%s attempt=%d state=%s err=%v", job.ID, job.UserID, result.attempt, result.state, err)
			_ = discord.Log(ctx, fmt.Sprintf("[Birthday Email Failed]\nJob: %s\nUser: %s\nEmail: %s\nAttempt: %d/%d\nState: %s\nError: %v", job.ID, job.UserID, maskEmail(job.User.Email), result.attempt, cfg.maxAttempts, result.state, err))
			continue
		}

		log.Printf("birthday email sent: job_id=%s user_id=%s", job.ID, job.UserID)
		_ = discord.Log(ctx, fmt.Sprintf("[Birthday Email Sent]\nJob: %s\nUser: %s\nEmail: %s", job.ID, job.UserID, maskEmail(job.User.Email)))
	}
}

type failedJobResult struct {
	state   string
	attempt int
}

type retryQueue interface {
	Enqueue(ctx context.Context, job domain.BirthdayJob) error
	EnqueueTo(ctx context.Context, name string, job domain.BirthdayJob) error
}

func handleFailedJob(ctx context.Context, redisQueue retryQueue, job domain.BirthdayJob, cfg senderConfig) (failedJobResult, error) {
	attempt := job.Attempt
	if attempt <= 0 {
		attempt = 1
	}

	if attempt >= cfg.maxAttempts {
		job.Attempt = attempt
		if err := redisQueue.EnqueueTo(ctx, cfg.deadLetterQueue, job); err != nil {
			return failedJobResult{}, err
		}
		return failedJobResult{state: "dead-letter", attempt: attempt}, nil
	}

	job.Attempt = attempt + 1
	if err := redisQueue.Enqueue(ctx, job); err != nil {
		return failedJobResult{}, err
	}
	return failedJobResult{state: "requeued", attempt: job.Attempt}, nil
}

func maskEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 || parts[0] == "" {
		return email
	}
	local := parts[0]
	if len(local) == 1 {
		return local[:1] + "***@" + parts[1]
	}
	return local[:1] + "***" + local[len(local)-1:] + "@" + parts[1]
}
