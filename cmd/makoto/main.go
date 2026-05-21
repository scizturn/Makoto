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
	"github.com/kyou-id/makoto/internal/email"
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

	runSender(ctx, redisQueue, processor, discord, cfg.RateLimitPerMinute)
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
	return client, client
}

func runSender(ctx context.Context, redisQueue *queue.RedisQueue, processor *worker.Processor, discord notify.DiscordLogger, rateLimitPerMinute int) {
	if rateLimitPerMinute <= 0 {
		rateLimitPerMinute = 100
	}
	interval := time.Minute / time.Duration(rateLimitPerMinute)
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
			log.Printf("birthday email failed: job_id=%s user_id=%s err=%v", job.ID, job.UserID, err)
			_ = discord.Log(ctx, fmt.Sprintf("[Birthday Email Failed]\nJob: %s\nUser: %s\nEmail: %s\nError: %v", job.ID, job.UserID, maskEmail(job.User.Email), err))
			continue
		}

		log.Printf("birthday email sent: job_id=%s user_id=%s", job.ID, job.UserID)
		_ = discord.Log(ctx, fmt.Sprintf("[Birthday Email Sent]\nJob: %s\nUser: %s\nEmail: %s", job.ID, job.UserID, maskEmail(job.User.Email)))
	}
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
