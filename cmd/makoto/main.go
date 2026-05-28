package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/kyou-id/makoto/internal/audit"
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
	auditLogger, err := buildAuditLogger(cfg)
	if err != nil {
		log.Fatalf("build audit logger: %v", err)
	}
	if auditLogger != nil {
		defer func() {
			if err := auditLogger.Close(); err != nil {
				log.Printf("audit db close failed: %v", err)
			}
		}()
	}
	processor.BeforeSend = func(ctx context.Context, job domain.BirthdayJob, result worker.ProcessResult) error {
		if err := auditLogger.MarkSending(ctx, auditInfoFromJob(job, result)); err != nil {
			log.Printf("birthday email audit sending update failed: job_id=%s user_id=%s err=%v", job.ID, job.UserID, err)
		}
		return nil
	}

	runSender(ctx, redisQueue, processor, discord, auditLogger, senderConfig{
		rateLimitPerMinute: cfg.RateLimitPerMinute,
		deadLetterQueue:    cfg.DeadLetterQueue,
		maxAttempts:        cfg.MaxAttempts,
	})
}

func buildAuditLogger(cfg config.Config) (*audit.Logger, error) {
	if strings.TrimSpace(cfg.DatabaseDSN) == "" {
		log.Print("OLD_DATABASE_* is empty; Makoto will run without email delivery audit logs")
		return nil, nil
	}
	return audit.Open(cfg.DatabaseDSN)
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
	retryBackoffs      []time.Duration
}

func runSender(ctx context.Context, redisQueue *queue.RedisQueue, processor *worker.Processor, discord notify.DiscordLogger, auditLogger *audit.Logger, cfg senderConfig) {
	if cfg.rateLimitPerMinute <= 0 {
		cfg.rateLimitPerMinute = 100
	}
	if cfg.maxAttempts <= 0 {
		cfg.maxAttempts = 3
	}
	if len(cfg.retryBackoffs) == 0 {
		cfg.retryBackoffs = []time.Duration{5 * time.Minute, 15 * time.Minute}
	}
	if recovered, err := redisQueue.RecoverProcessing(ctx); err != nil {
		log.Printf("redis processing recovery failed: %v", err)
	} else if recovered > 0 {
		log.Printf("redis processing recovery requeued %d job(s)", recovered)
	}
	interval := time.Minute / time.Duration(cfg.rateLimitPerMinute)
	if interval <= 0 {
		interval = time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		processingJob, err := redisQueue.Dequeue(ctx, 5*time.Second)
		if errors.Is(err, redis.Nil) {
			continue
		}
		if err != nil {
			log.Printf("redis dequeue failed: %v", err)
			continue
		}
		job := processingJob.Job

		select {
		case <-ctx.Done():
			log.Print(ctx.Err())
			return
		case <-ticker.C:
		}

		processResult, err := processor.Process(ctx, job)
		auditInfo := auditInfoFromJob(job, processResult)
		if err != nil {
			result, failureErr := handleFailedJob(ctx, redisQueue, auditLogger, auditInfo, job, err, cfg)
			if failureErr != nil {
				log.Printf("birthday email failure handling failed: job_id=%s user_id=%s err=%v failure_err=%v", job.ID, job.UserID, err, failureErr)
				_ = discord.Log(ctx, fmt.Sprintf("[Birthday Email Failure Handling Failed]\nJob: %s\nUser: %s\nEmail: %s\nError: %v\nFailure handling error: %v", job.ID, job.UserID, maskEmail(job.User.Email), err, failureErr))
				continue
			}
			if ackErr := redisQueue.Ack(ctx, processingJob.Payload); ackErr != nil {
				log.Printf("birthday email failed but processing ack failed: job_id=%s user_id=%s err=%v ack_err=%v", job.ID, job.UserID, err, ackErr)
				_ = discord.Log(ctx, fmt.Sprintf("[Birthday Email Processing Ack Failed]\nJob: %s\nUser: %s\nEmail: %s\nError: %v\nAck error: %v", job.ID, job.UserID, maskEmail(job.User.Email), err, ackErr))
				continue
			}
			log.Printf("birthday email failed: job_id=%s user_id=%s attempt=%d state=%s err=%v", job.ID, job.UserID, result.attempt, result.state, err)
			_ = discord.Log(ctx, fmt.Sprintf("[Birthday Email Failed]\nJob: %s\nUser: %s\nEmail: %s\nAttempt: %d/%d\nState: %s\nRetry delay: %s\nError: %v", job.ID, job.UserID, maskEmail(job.User.Email), result.attempt, cfg.maxAttempts, result.state, result.delay, err))
			continue
		}

		if err := auditLogger.MarkSent(ctx, auditInfo); err != nil {
			log.Printf("birthday email audit sent update failed: job_id=%s user_id=%s err=%v", job.ID, job.UserID, err)
		}
		if err := redisQueue.Ack(ctx, processingJob.Payload); err != nil {
			log.Printf("birthday email sent but processing ack failed: job_id=%s user_id=%s err=%v", job.ID, job.UserID, err)
			_ = discord.Log(ctx, fmt.Sprintf("[Birthday Email Processing Ack Failed]\nJob: %s\nUser: %s\nEmail: %s\nAck error: %v", job.ID, job.UserID, maskEmail(job.User.Email), err))
			continue
		}
		log.Printf("birthday email sent: job_id=%s user_id=%s", job.ID, job.UserID)
		_ = discord.Log(ctx, fmt.Sprintf("[Birthday Email Sent]\nJob: %s\nUser: %s\nEmail: %s", job.ID, job.UserID, maskEmail(job.User.Email)))
	}
}

type failedJobResult struct {
	state   string
	attempt int
	delay   time.Duration
}

type retryQueue interface {
	Enqueue(ctx context.Context, job domain.BirthdayJob) error
	EnqueueTo(ctx context.Context, name string, job domain.BirthdayJob) error
}

func handleFailedJob(ctx context.Context, redisQueue retryQueue, auditLogger *audit.Logger, auditInfo audit.MessageInfo, job domain.BirthdayJob, processErr error, cfg senderConfig) (failedJobResult, error) {
	attempt := job.Attempt
	if attempt <= 0 {
		attempt = 1
	}
	auditInfo.Attempt = attempt
	auditInfo.FailureReason = processErr.Error()

	if attempt >= cfg.maxAttempts {
		job.Attempt = attempt
		if err := redisQueue.EnqueueTo(ctx, cfg.deadLetterQueue, job); err != nil {
			return failedJobResult{}, err
		}
		if err := auditLogger.MarkDeadLetter(ctx, auditInfo); err != nil {
			return failedJobResult{}, err
		}
		return failedJobResult{state: "dead-letter", attempt: attempt}, nil
	}

	if err := auditLogger.MarkFailed(ctx, auditInfo); err != nil {
		return failedJobResult{}, err
	}
	job.Attempt = attempt + 1
	delay := retryDelay(attempt, cfg.retryBackoffs)
	if delay > 0 {
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return failedJobResult{}, ctx.Err()
		case <-timer.C:
		}
	}
	if err := redisQueue.Enqueue(ctx, job); err != nil {
		return failedJobResult{}, err
	}
	if err := auditLogger.InsertRetryQueued(ctx, job.ID, attempt); err != nil {
		return failedJobResult{}, err
	}
	return failedJobResult{state: "requeued", attempt: job.Attempt, delay: delay}, nil
}

func auditInfoFromJob(job domain.BirthdayJob, result worker.ProcessResult) audit.MessageInfo {
	attempt := job.Attempt
	if attempt <= 0 {
		attempt = 1
	}
	return audit.MessageInfo{
		JobID:             job.ID,
		Attempt:           attempt,
		TemplateID:        result.TemplateID,
		Subject:           result.Subject,
		ActionURL:         result.ActionURL,
		ProviderMessageID: result.SendResult.MessageID,
		ProviderStatus:    result.SendResult.StatusCode,
		ProviderResponse:  result.SendResult.Response,
	}
}

func retryDelay(attempt int, backoffs []time.Duration) time.Duration {
	if attempt <= 0 || len(backoffs) == 0 {
		return 0
	}
	index := attempt - 1
	if index >= len(backoffs) {
		index = len(backoffs) - 1
	}
	return backoffs[index]
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
