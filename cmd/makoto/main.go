package main

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
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

	// --- Birthday ---
	birthdayCampaign := campaign.BirthdayCampaign{
		TemplateIDs: cfg.TemplateIDs,
		Subjects:    cfg.EmailSubjects,
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

	// --- Anniversary ---
	anniversaryCampaign := campaign.AnniversaryCampaign{
		TemplateIDs: cfg.AnniversaryTemplateIDs,
		Subjects:    cfg.AnniversaryEmailSubjects,
		Closing:     "Terima kasih sudah menjadi bagian dari Kyou! 🎉",
		ActionURL:   cfg.ActionURL,
	}
	anniversaryProcessor := worker.NewAnniversaryProcessor(nil, sender, validator, voucherIssuer, anniversaryCampaign)
	anniversaryProcessor.Domain = cfg.KirimEmailDomain
	anniversaryProcessor.FromEmail = cfg.FromEmail
	anniversaryProcessor.FromName = cfg.FromName
	if cfg.AnniversaryEmailTemplateDir != "" {
		anniversaryProcessor.Renderer = emailtemplate.FileRenderer{
			Dir:     cfg.AnniversaryEmailTemplateDir,
			Subject: cfg.AnniversaryEmailSubject,
		}
	}

	// --- Discounted Wishlist ---
	discountedWishlistCampaign := campaign.DiscountedWishlistCampaign{
		TemplateIDs:    cfg.DiscountedWishlistTemplateIDs,
		Subjects:       cfg.DiscountedWishlistEmailSubjects,
		Greetings:      cfg.DiscountedWishlistGreetings,
		WishlistURL:    cfg.DiscountedWishlistURL,
		Closing:        "Yuk cek wishlistmu di Kyou sekarang!",
	}
	discountedWishlistProcessor := worker.NewDiscountedWishlistProcessor(sender, validator, discountedWishlistCampaign)
	discountedWishlistProcessor.Domain = cfg.KirimEmailDomain
	discountedWishlistProcessor.FromEmail = cfg.FromEmail
	discountedWishlistProcessor.FromName = cfg.FromName
	if cfg.DiscountedWishlistEmailTemplateDir != "" {
		discountedWishlistProcessor.Renderer = emailtemplate.FileRenderer{
			Dir:     cfg.DiscountedWishlistEmailTemplateDir,
			Subject: cfg.DiscountedWishlistEmailSubject,
		}
	}

	// --- PO Ready ---
	poReadyCampaign := campaign.PoReadyCampaign{
		TemplateIDs: cfg.PoReadyTemplateIDs,
		Subjects:    cfg.PoReadyEmailSubjects,
		Greetings:   cfg.PoReadyGreetings,
		WishlistURL: cfg.PoReadyURL,
		Closing:     "Stok ready biasanya cepat habis — cek wishlist kamu sebelum keduluan!",
	}
	poReadyProcessor := worker.NewPoReadyProcessor(sender, validator, poReadyCampaign)
	poReadyProcessor.Domain = cfg.KirimEmailDomain
	poReadyProcessor.FromEmail = cfg.FromEmail
	poReadyProcessor.FromName = cfg.FromName
	if cfg.PoReadyEmailTemplateDir != "" {
		poReadyProcessor.Renderer = emailtemplate.FileRenderer{
			Dir:     cfg.PoReadyEmailTemplateDir,
			Subject: cfg.PoReadyEmailSubject,
		}
	}

	// --- Wishlist Back In ---
	wishlistBackInCampaign := campaign.WishlistBackInCampaign{
		TemplateIDs: cfg.WishlistBackInTemplateIDs,
		Subjects:    cfg.WishlistBackInEmailSubjects,
		Greetings:   cfg.WishlistBackInGreetings,
		ActionURL:   cfg.WishlistBackInActionURL,
		WishlistURL: "https://kyou.id/user/wishlist",
		Closing:     "Jangan sampai kelewatan lagi ya!",
	}
	wishlistBackInProcessor := worker.NewWishlistBackInProcessor(sender, validator, wishlistBackInCampaign)
	wishlistBackInProcessor.Domain = cfg.KirimEmailDomain
	wishlistBackInProcessor.FromEmail = cfg.FromEmail
	wishlistBackInProcessor.FromName = cfg.FromName
	if cfg.WishlistBackInEmailTemplateDir != "" {
		wishlistBackInProcessor.Renderer = emailtemplate.FileRenderer{Dir: cfg.WishlistBackInEmailTemplateDir, Subject: cfg.WishlistBackInEmailSubject}
	}

	// --- Winback ---
	winbackCampaign := campaign.WinbackCampaign{
		TemplateIDs: cfg.WinbackTemplateIDs,
		Subjects:    cfg.WinbackEmailSubjects,
		Greetings:   cfg.WinbackGreetings,
		ActionURL:   cfg.WinbackActionURL,
		Closing:     "Ayo #RayakanHobimu kembali bersama Kyou!",
	}
	winbackProcessor := worker.NewWinbackProcessor(sender, validator, winbackCampaign)
	winbackProcessor.Domain = cfg.KirimEmailDomain
	winbackProcessor.FromEmail = cfg.FromEmail
	winbackProcessor.FromName = cfg.FromName
	if cfg.WinbackEmailTemplateDir != "" {
		winbackProcessor.Renderer = emailtemplate.FileRenderer{
			Dir:     cfg.WinbackEmailTemplateDir,
			Subject: cfg.WinbackEmailSubject,
		}
	}

	// --- Leftover Cart ---
	leftoverCartCampaign := campaign.LeftoverCartCampaign{
		TemplateIDs: cfg.LeftoverCartTemplateIDs,
		Greetings:   cfg.LeftoverCartGreetings,
		CartURL:     cfg.LeftoverCartURL,
		Closing:     "Sampai ketemu lagi di Kyou!",
	}
	leftoverCartProcessor := worker.NewLeftoverCartProcessor(sender, validator, leftoverCartCampaign)
	leftoverCartProcessor.Domain = cfg.KirimEmailDomain
	leftoverCartProcessor.FromEmail = cfg.FromEmail
	leftoverCartProcessor.FromName = cfg.FromName
	if cfg.LeftoverCartEmailTemplateDir != "" {
		leftoverCartProcessor.Renderer = emailtemplate.FileRenderer{
			Dir:     cfg.LeftoverCartEmailTemplateDir,
			Subject: cfg.LeftoverCartEmailSubject,
		}
	}

	// --- Queues ---
	redisQueue := queue.NewRedisQueue(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB, cfg.QueueName)
	defer func() {
		if err := redisQueue.Close(); err != nil {
			log.Printf("redis close failed: %v", err)
		}
	}()

	anniversaryQueue := queue.NewAnniversaryRedisQueue(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB, cfg.AnniversaryQueueName)
	defer func() {
		if err := anniversaryQueue.Close(); err != nil {
			log.Printf("anniversary redis close failed: %v", err)
		}
	}()

	leftoverCartQueue := queue.NewLeftoverCartRedisQueue(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB, cfg.LeftoverCartQueueName)
	defer func() {
		if err := leftoverCartQueue.Close(); err != nil {
			log.Printf("leftover cart redis close failed: %v", err)
		}
	}()

	discountedWishlistQueue := queue.NewDiscountedWishlistRedisQueue(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB, cfg.DiscountedWishlistQueueName)
	defer func() {
		if err := discountedWishlistQueue.Close(); err != nil {
			log.Printf("discounted wishlist redis close failed: %v", err)
		}
	}()

	wishlistBackInQueue := queue.NewWishlistBackInRedisQueue(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB, cfg.WishlistBackInQueueName)
	defer func() {
		if err := wishlistBackInQueue.Close(); err != nil {
			log.Printf("wishlist back in redis close failed: %v", err)
		}
	}()

	winbackQueue := queue.NewWinbackRedisQueue(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB, cfg.WinbackQueueName)
	defer func() {
		if err := winbackQueue.Close(); err != nil {
			log.Printf("winback redis close failed: %v", err)
		}
	}()

	poReadyQueue := queue.NewPoReadyRedisQueue(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB, cfg.PoReadyQueueName)
	defer func() {
		if err := poReadyQueue.Close(); err != nil {
			log.Printf("po ready redis close failed: %v", err)
		}
	}()

	// --- Discord + Audit ---
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
	anniversaryProcessor.BeforeSend = func(ctx context.Context, job domain.AnniversaryJob, result worker.ProcessResult) error {
		if err := auditLogger.MarkSending(ctx, auditInfoFromAnniversaryJob(job, result)); err != nil {
			log.Printf("anniversary email audit sending update failed: job_id=%s user_id=%s err=%v", job.ID, job.UserID, err)
		}
		return nil
	}
	leftoverCartProcessor.BeforeSend = func(ctx context.Context, job domain.LeftoverCartJob, result worker.ProcessResult) error {
		if err := auditLogger.MarkSending(ctx, auditInfoFromLeftoverCartJob(job, result)); err != nil {
			log.Printf("leftover cart email audit sending update failed: job_id=%s user_id=%s err=%v", job.ID, job.UserID, err)
		}
		return nil
	}
	discountedWishlistProcessor.BeforeSend = func(ctx context.Context, job domain.DiscountedWishlistJob, result worker.ProcessResult) error {
		if err := auditLogger.MarkSending(ctx, auditInfoFromDiscountedWishlistJob(job, result)); err != nil {
			log.Printf("discounted wishlist email audit sending update failed: job_id=%s user_id=%s err=%v", job.ID, job.UserID, err)
		}
		return nil
	}
	wishlistBackInProcessor.BeforeSend = func(ctx context.Context, job domain.WishlistBackInJob, result worker.ProcessResult) error {
		if err := auditLogger.MarkSending(ctx, auditInfoFromWishlistBackInJob(job, result)); err != nil {
			log.Printf("wishlist back in email audit sending update failed: job_id=%s user_id=%s err=%v", job.ID, job.UserID, err)
		}
		return nil
	}
	winbackProcessor.BeforeSend = func(ctx context.Context, job domain.WinbackJob, result worker.ProcessResult) error {
		if err := auditLogger.MarkSending(ctx, auditInfoFromWinbackJob(job, result)); err != nil {
			log.Printf("winback email audit sending update failed: job_id=%s user_id=%s err=%v", job.ID, job.UserID, err)
		}
		return nil
	}
	poReadyProcessor.BeforeSend = func(ctx context.Context, job domain.PoReadyJob, result worker.ProcessResult) error {
		if err := auditLogger.MarkSending(ctx, auditInfoFromPoReadyJob(job, result)); err != nil {
			log.Printf("po ready email audit sending update failed: job_id=%s user_id=%s err=%v", job.ID, job.UserID, err)
		}
		return nil
	}

	// --- Run both workers concurrently ---
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		runSender(ctx, redisQueue, processor, discord, auditLogger, senderConfig{
			rateLimitPerMinute: cfg.RateLimitPerMinute,
			deadLetterQueue:    cfg.DeadLetterQueue,
			maxAttempts:        cfg.MaxAttempts,
		})
	}()

	if cfg.AnniversaryEnabled {
		wg.Add(1)
		go func() {
			defer wg.Done()
			runAnniversarySender(ctx, anniversaryQueue, anniversaryProcessor, discord, auditLogger, senderConfig{
				rateLimitPerMinute: cfg.RateLimitPerMinute,
				deadLetterQueue:    cfg.AnniversaryDeadLetterQueue,
				maxAttempts:        cfg.MaxAttempts,
			})
		}()
	} else {
		log.Print("MAKOTO_ANNIVERSARY_ENABLED is false; skipping anniversary sender")
	}

	if cfg.LeftoverCartEnabled {
		wg.Add(1)
		go func() {
			defer wg.Done()
			runLeftoverCartSender(ctx, leftoverCartQueue, leftoverCartProcessor, discord, auditLogger, senderConfig{
				rateLimitPerMinute: cfg.RateLimitPerMinute,
				deadLetterQueue:    cfg.LeftoverCartDeadLetterQueue,
				maxAttempts:        cfg.MaxAttempts,
			})
		}()
	} else {
		log.Print("MAKOTO_LEFTOVER_CART_ENABLED is false; skipping leftover cart sender")
	}

	if cfg.DiscountedWishlistEnabled {
		wg.Add(1)
		go func() {
			defer wg.Done()
			runDiscountedWishlistSender(ctx, discountedWishlistQueue, discountedWishlistProcessor, discord, auditLogger, senderConfig{
				rateLimitPerMinute: cfg.RateLimitPerMinute,
				deadLetterQueue:    cfg.DiscountedWishlistDeadLetterQueue,
				maxAttempts:        cfg.MaxAttempts,
			})
		}()
	} else {
		log.Print("MAKOTO_DISCOUNTED_WISHLIST_ENABLED is false; skipping discounted wishlist sender")
	}

	if cfg.WishlistBackInEnabled {
		wg.Add(1)
		go func() {
			defer wg.Done()
			runWishlistBackInSender(ctx, wishlistBackInQueue, wishlistBackInProcessor, discord, auditLogger, senderConfig{
				rateLimitPerMinute: cfg.RateLimitPerMinute,
				deadLetterQueue:    cfg.WishlistBackInDeadLetterQueue,
				maxAttempts:        cfg.MaxAttempts,
			})
		}()
	} else {
		log.Print("MAKOTO_WISHLIST_BACK_IN_ENABLED is false; skipping wishlist back in sender")
	}

	if cfg.WinbackEnabled {
		wg.Add(1)
		go func() {
			defer wg.Done()
			runWinbackSender(ctx, winbackQueue, winbackProcessor, discord, auditLogger, senderConfig{
				rateLimitPerMinute: cfg.RateLimitPerMinute,
				deadLetterQueue:    cfg.WinbackDeadLetterQueue,
				maxAttempts:        cfg.MaxAttempts,
			})
		}()
	} else {
		log.Print("MAKOTO_WINBACK_ENABLED is false; skipping winback sender")
	}

	if cfg.PoReadyEnabled {
		wg.Add(1)
		go func() {
			defer wg.Done()
			runPoReadySender(ctx, poReadyQueue, poReadyProcessor, discord, auditLogger, senderConfig{
				rateLimitPerMinute: cfg.RateLimitPerMinute,
				deadLetterQueue:    cfg.PoReadyDeadLetterQueue,
				maxAttempts:        cfg.MaxAttempts,
			})
		}()
	} else {
		log.Print("MAKOTO_PO_READY_ENABLED is false; skipping po ready sender")
	}
	wg.Wait()
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
	log.Printf("Kirim.email config: base_url=%s domain=%s validate=%t username=%s token_len=%d token_sha256=%s",
		cfg.KirimEmailBaseURL,
		cfg.KirimEmailDomain,
		cfg.KirimEmailValidate,
		redactForLog(cfg.KirimEmailUsername),
		len(cfg.KirimEmailAPIToken),
		shortSHA256(cfg.KirimEmailAPIToken),
	)

	client := email.KirimClient{
		BaseURL:  cfg.KirimEmailBaseURL,
		Username: cfg.KirimEmailUsername,
		APIToken: cfg.KirimEmailAPIToken,
	}
	if !cfg.KirimEmailValidate {
		log.Print("KIRIM_EMAIL_VALIDATE is false; skipping strict email validation")
		return client, email.AllowAllValidator{}
	}
	if cfg.KirimEmailValidationUsername == "" || cfg.KirimEmailValidationAPIToken == "" {
		log.Print("Kirim.email validation credentials are empty; using send credentials for strict email validation")
		return client, client
	}
	log.Printf("Kirim.email validation config: username=%s token_len=%d token_sha256=%s",
		redactForLog(cfg.KirimEmailValidationUsername),
		len(cfg.KirimEmailValidationAPIToken),
		shortSHA256(cfg.KirimEmailValidationAPIToken),
	)
	var validator email.Validator = email.KirimClient{
		BaseURL:  cfg.KirimEmailBaseURL,
		Username: cfg.KirimEmailValidationUsername,
		APIToken: cfg.KirimEmailValidationAPIToken,
	}
	if cfg.KirimEmailValidationFailOpen {
		log.Print("KIRIM_EMAIL_VALIDATION_FAIL_OPEN is true; validation provider errors will not block sending")
		validator = email.FailOpenValidator{Validator: validator}
	}
	return client, validator
}

func redactForLog(value string) string {
	if value == "" {
		return "<empty>"
	}
	if len(value) <= 8 {
		return fmt.Sprintf("<redacted len=%d>", len(value))
	}
	return fmt.Sprintf("%s...%s(len=%d)", value[:4], value[len(value)-4:], len(value))
}

func shortSHA256(value string) string {
	if value == "" {
		return "<empty>"
	}
	sum := sha256.Sum256([]byte(value))
	return fmt.Sprintf("%x", sum[:4])
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

func runAnniversarySender(ctx context.Context, redisQueue *queue.AnniversaryRedisQueue, processor *worker.AnniversaryProcessor, discord notify.DiscordLogger, auditLogger *audit.Logger, cfg senderConfig) {
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
		log.Printf("anniversary redis processing recovery failed: %v", err)
	} else if recovered > 0 {
		log.Printf("anniversary redis processing recovery requeued %d job(s)", recovered)
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
			log.Printf("anniversary redis dequeue failed: %v", err)
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
		auditInfo := auditInfoFromAnniversaryJob(job, processResult)
		if err != nil {
			log.Printf("anniversary email failed: job_id=%s user_id=%s err=%v", job.ID, job.UserID, err)
			_ = discord.Log(ctx, fmt.Sprintf("[Anniversary Email Failed]\nJob: %s\nUser: %s\nEmail: %s\nError: %v", job.ID, job.UserID, maskEmail(job.User.Email), err))
			if ackErr := redisQueue.Ack(ctx, processingJob.Payload); ackErr != nil {
				log.Printf("anniversary email failed but ack failed: job_id=%s err=%v ack_err=%v", job.ID, err, ackErr)
			}
			if err := auditLogger.MarkFailed(ctx, auditInfo); err != nil {
				log.Printf("anniversary email audit failed update failed: job_id=%s err=%v", job.ID, err)
			}
			continue
		}

		if err := auditLogger.MarkSent(ctx, auditInfo); err != nil {
			log.Printf("anniversary email audit sent update failed: job_id=%s user_id=%s err=%v", job.ID, job.UserID, err)
		}
		if err := redisQueue.Ack(ctx, processingJob.Payload); err != nil {
			log.Printf("anniversary email sent but processing ack failed: job_id=%s user_id=%s err=%v", job.ID, job.UserID, err)
			_ = discord.Log(ctx, fmt.Sprintf("[Anniversary Email Processing Ack Failed]\nJob: %s\nUser: %s\nEmail: %s\nAck error: %v", job.ID, job.UserID, maskEmail(job.User.Email), err))
			continue
		}
		log.Printf("anniversary email sent: job_id=%s user_id=%s", job.ID, job.UserID)
		_ = discord.Log(ctx, fmt.Sprintf("[Anniversary Email Sent]\nJob: %s\nUser: %s\nEmail: %s", job.ID, job.UserID, maskEmail(job.User.Email)))
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
		if err := auditLogger.MarkDeadLetter(ctx, auditInfo); err != nil {
			return failedJobResult{}, err
		}
		if err := redisQueue.EnqueueTo(ctx, cfg.deadLetterQueue, job); err != nil {
			failedAuditInfo := auditInfo
			failedAuditInfo.FailureReason = "dead-letter enqueue failed: " + err.Error()
			return failedJobResult{}, errors.Join(err, auditLogger.MarkFailed(ctx, failedAuditInfo))
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
	if err := auditLogger.InsertRetryQueued(ctx, job.ID, attempt); err != nil {
		return failedJobResult{}, err
	}
	if err := redisQueue.Enqueue(ctx, job); err != nil {
		retryAuditInfo := auditInfo
		retryAuditInfo.Attempt = job.Attempt
		retryAuditInfo.FailureReason = "retry enqueue failed: " + err.Error()
		return failedJobResult{}, errors.Join(err, auditLogger.MarkFailed(ctx, retryAuditInfo))
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

func auditInfoFromAnniversaryJob(job domain.AnniversaryJob, result worker.ProcessResult) audit.MessageInfo {
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

func auditInfoFromPoReadyJob(job domain.PoReadyJob, result worker.ProcessResult) audit.MessageInfo {
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

func runPoReadySender(ctx context.Context, q *queue.PoReadyRedisQueue, processor *worker.PoReadyProcessor, discord notify.DiscordLogger, auditLogger *audit.Logger, cfg senderConfig) {
	if cfg.rateLimitPerMinute <= 0 {
		cfg.rateLimitPerMinute = 100
	}
	if cfg.maxAttempts <= 0 {
		cfg.maxAttempts = 3
	}
	if len(cfg.retryBackoffs) == 0 {
		cfg.retryBackoffs = []time.Duration{5 * time.Minute, 15 * time.Minute}
	}
	if recovered, err := q.RecoverProcessing(ctx); err != nil {
		log.Printf("po ready redis processing recovery failed: %v", err)
	} else if recovered > 0 {
		log.Printf("po ready redis processing recovery requeued %d job(s)", recovered)
	}
	interval := time.Minute / time.Duration(cfg.rateLimitPerMinute)
	if interval <= 0 {
		interval = time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		processingJob, err := q.Dequeue(ctx, 5*time.Second)
		if errors.Is(err, redis.Nil) {
			continue
		}
		if err != nil {
			log.Printf("po ready redis dequeue failed: %v", err)
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
		auditInfo := auditInfoFromPoReadyJob(job, processResult)
		if err != nil {
			result, failureErr := handleFailedPoReadyJob(ctx, q, auditLogger, auditInfo, job, err, cfg)
			if failureErr != nil {
				log.Printf("po ready failure handling failed: job_id=%s user_id=%s err=%v failure_err=%v", job.ID, job.UserID, err, failureErr)
				_ = discord.Log(ctx, fmt.Sprintf("[PO Ready Failure Handling Failed]\nJob: %s\nUser: %s\nEmail: %s\nError: %v\nFailure handling error: %v", job.ID, job.UserID, maskEmail(job.User.Email), err, failureErr))
				continue
			}
			if ackErr := q.Ack(ctx, processingJob.Payload); ackErr != nil {
				log.Printf("po ready failed but processing ack failed: job_id=%s user_id=%s err=%v ack_err=%v", job.ID, job.UserID, err, ackErr)
				continue
			}
			log.Printf("po ready email failed: job_id=%s user_id=%s attempt=%d state=%s err=%v", job.ID, job.UserID, result.attempt, result.state, err)
			_ = discord.Log(ctx, fmt.Sprintf("[PO Ready Email Failed]\nJob: %s\nUser: %s\nEmail: %s\nAttempt: %d/%d\nState: %s\nRetry delay: %s\nError: %v", job.ID, job.UserID, maskEmail(job.User.Email), result.attempt, cfg.maxAttempts, result.state, result.delay, err))
			continue
		}
		if processResult.Outcome == worker.ProcessOutcomeSkipped {
			if err := auditLogger.MarkSkipped(ctx, auditInfo, processResult.SkipReason); err != nil {
				log.Printf("po ready email audit skipped update failed: job_id=%s user_id=%s err=%v", job.ID, job.UserID, err)
			}
			if err := q.Ack(ctx, processingJob.Payload); err != nil {
				log.Printf("po ready email skipped but ack failed: job_id=%s user_id=%s err=%v", job.ID, job.UserID, err)
				continue
			}
			log.Printf("po ready email skipped: job_id=%s user_id=%s reason=%s", job.ID, job.UserID, processResult.SkipReason)
			continue
		}

		if err := auditLogger.MarkSent(ctx, auditInfo); err != nil {
			log.Printf("po ready email audit sent update failed: job_id=%s user_id=%s err=%v", job.ID, job.UserID, err)
		}
		if err := q.Ack(ctx, processingJob.Payload); err != nil {
			log.Printf("po ready email sent but ack failed: job_id=%s user_id=%s err=%v", job.ID, job.UserID, err)
			_ = discord.Log(ctx, fmt.Sprintf("[PO Ready Email Ack Failed]\nJob: %s\nUser: %s\nEmail: %s\nAck error: %v", job.ID, job.UserID, maskEmail(job.User.Email), err))
			continue
		}
		log.Printf("po ready email sent: job_id=%s user_id=%s items=%d", job.ID, job.UserID, len(job.Items))
		_ = discord.Log(ctx, fmt.Sprintf("[PO Ready Email Sent]\nJob: %s\nUser: %s\nEmail: %s\nItems: %d", job.ID, job.UserID, maskEmail(job.User.Email), len(job.Items)))
	}
}

type poReadyRetryQueue interface {
	Enqueue(ctx context.Context, job domain.PoReadyJob) error
	EnqueueTo(ctx context.Context, name string, job domain.PoReadyJob) error
}

func handleFailedPoReadyJob(ctx context.Context, redisQueue poReadyRetryQueue, auditLogger *audit.Logger, auditInfo audit.MessageInfo, job domain.PoReadyJob, processErr error, cfg senderConfig) (failedJobResult, error) {
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

func auditInfoFromDiscountedWishlistJob(job domain.DiscountedWishlistJob, result worker.ProcessResult) audit.MessageInfo {
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

func auditInfoFromWishlistBackInJob(job domain.WishlistBackInJob, result worker.ProcessResult) audit.MessageInfo {
	attempt := job.Attempt
	if attempt <= 0 {
		attempt = 1
	}
	return audit.MessageInfo{
		JobID: job.ID, Attempt: attempt, TemplateID: result.TemplateID, Subject: result.Subject,
		ActionURL: result.ActionURL, ProviderMessageID: result.SendResult.MessageID,
		ProviderStatus: result.SendResult.StatusCode, ProviderResponse: result.SendResult.Response,
	}
}

func auditInfoFromLeftoverCartJob(job domain.LeftoverCartJob, result worker.ProcessResult) audit.MessageInfo {
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

func runLeftoverCartSender(ctx context.Context, q *queue.LeftoverCartRedisQueue, processor *worker.LeftoverCartProcessor, discord notify.DiscordLogger, auditLogger *audit.Logger, cfg senderConfig) {
	if cfg.rateLimitPerMinute <= 0 {
		cfg.rateLimitPerMinute = 100
	}
	if cfg.maxAttempts <= 0 {
		cfg.maxAttempts = 3
	}
	if recovered, err := q.RecoverProcessing(ctx); err != nil {
		log.Printf("leftover cart redis processing recovery failed: %v", err)
	} else if recovered > 0 {
		log.Printf("leftover cart redis processing recovery requeued %d job(s)", recovered)
	}
	interval := time.Minute / time.Duration(cfg.rateLimitPerMinute)
	if interval <= 0 {
		interval = time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		processingJob, err := q.Dequeue(ctx, 5*time.Second)
		if errors.Is(err, redis.Nil) {
			continue
		}
		if err != nil {
			log.Printf("leftover cart redis dequeue failed: %v", err)
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
		auditInfo := auditInfoFromLeftoverCartJob(job, processResult)
		if err != nil {
			log.Printf("leftover cart email failed: job_id=%s user_id=%s err=%v", job.ID, job.UserID, err)
			_ = discord.Log(ctx, fmt.Sprintf("[Leftover Cart Email Failed]\nJob: %s\nUser: %s\nEmail: %s\nError: %v", job.ID, job.UserID, maskEmail(job.User.Email), err))
			if ackErr := q.Ack(ctx, processingJob.Payload); ackErr != nil {
				log.Printf("leftover cart email failed but ack failed: job_id=%s err=%v ack_err=%v", job.ID, err, ackErr)
			}
			if err := auditLogger.MarkFailed(ctx, auditInfo); err != nil {
				log.Printf("leftover cart email audit failed update failed: job_id=%s err=%v", job.ID, err)
			}
			continue
		}

		if err := auditLogger.MarkSent(ctx, auditInfo); err != nil {
			log.Printf("leftover cart email audit sent update failed: job_id=%s user_id=%s err=%v", job.ID, job.UserID, err)
		}
		if err := q.Ack(ctx, processingJob.Payload); err != nil {
			log.Printf("leftover cart email sent but ack failed: job_id=%s user_id=%s err=%v", job.ID, job.UserID, err)
			_ = discord.Log(ctx, fmt.Sprintf("[Leftover Cart Email Ack Failed]\nJob: %s\nUser: %s\nEmail: %s\nAck error: %v", job.ID, job.UserID, maskEmail(job.User.Email), err))
			continue
		}
		log.Printf("leftover cart email sent: job_id=%s user_id=%s", job.ID, job.UserID)
		_ = discord.Log(ctx, fmt.Sprintf("[Leftover Cart Email Sent]\nJob: %s\nUser: %s\nEmail: %s", job.ID, job.UserID, maskEmail(job.User.Email)))
	}
}

func runDiscountedWishlistSender(ctx context.Context, q *queue.DiscountedWishlistRedisQueue, processor *worker.DiscountedWishlistProcessor, discord notify.DiscordLogger, auditLogger *audit.Logger, cfg senderConfig) {
	if cfg.rateLimitPerMinute <= 0 {
		cfg.rateLimitPerMinute = 100
	}
	if cfg.maxAttempts <= 0 {
		cfg.maxAttempts = 3
	}
	if len(cfg.retryBackoffs) == 0 {
		cfg.retryBackoffs = []time.Duration{5 * time.Minute, 15 * time.Minute}
	}
	if recovered, err := q.RecoverProcessing(ctx); err != nil {
		log.Printf("discounted wishlist redis processing recovery failed: %v", err)
	} else if recovered > 0 {
		log.Printf("discounted wishlist redis processing recovery requeued %d job(s)", recovered)
	}
	interval := time.Minute / time.Duration(cfg.rateLimitPerMinute)
	if interval <= 0 {
		interval = time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		processingJob, err := q.Dequeue(ctx, 5*time.Second)
		if errors.Is(err, redis.Nil) {
			continue
		}
		if err != nil {
			log.Printf("discounted wishlist redis dequeue failed: %v", err)
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
		auditInfo := auditInfoFromDiscountedWishlistJob(job, processResult)
		if err != nil {
			result, failureErr := handleFailedDiscountedWishlistJob(ctx, q, auditLogger, auditInfo, job, err, cfg)
			if failureErr != nil {
				log.Printf("discounted wishlist failure handling failed: job_id=%s user_id=%s err=%v failure_err=%v", job.ID, job.UserID, err, failureErr)
				_ = discord.Log(ctx, fmt.Sprintf("[Discounted Wishlist Failure Handling Failed]\nJob: %s\nUser: %s\nEmail: %s\nError: %v\nFailure handling error: %v", job.ID, job.UserID, maskEmail(job.User.Email), err, failureErr))
				continue
			}
			if ackErr := q.Ack(ctx, processingJob.Payload); ackErr != nil {
				log.Printf("discounted wishlist failed but processing ack failed: job_id=%s user_id=%s err=%v ack_err=%v", job.ID, job.UserID, err, ackErr)
				continue
			}
			log.Printf("discounted wishlist email failed: job_id=%s user_id=%s attempt=%d state=%s err=%v", job.ID, job.UserID, result.attempt, result.state, err)
			_ = discord.Log(ctx, fmt.Sprintf("[Discounted Wishlist Email Failed]\nJob: %s\nUser: %s\nEmail: %s\nAttempt: %d/%d\nState: %s\nRetry delay: %s\nError: %v", job.ID, job.UserID, maskEmail(job.User.Email), result.attempt, cfg.maxAttempts, result.state, result.delay, err))
			continue
		}
		if processResult.Outcome == worker.ProcessOutcomeSkipped {
			if err := auditLogger.MarkSkipped(ctx, auditInfo, processResult.SkipReason); err != nil {
				log.Printf("discounted wishlist email audit skipped update failed: job_id=%s user_id=%s err=%v", job.ID, job.UserID, err)
			}
			if err := q.Ack(ctx, processingJob.Payload); err != nil {
				log.Printf("discounted wishlist email skipped but ack failed: job_id=%s user_id=%s err=%v", job.ID, job.UserID, err)
				continue
			}
			log.Printf("discounted wishlist email skipped: job_id=%s user_id=%s reason=%s", job.ID, job.UserID, processResult.SkipReason)
			continue
		}

		if err := auditLogger.MarkSent(ctx, auditInfo); err != nil {
			log.Printf("discounted wishlist email audit sent update failed: job_id=%s user_id=%s err=%v", job.ID, job.UserID, err)
		}
		if err := q.Ack(ctx, processingJob.Payload); err != nil {
			log.Printf("discounted wishlist email sent but ack failed: job_id=%s user_id=%s err=%v", job.ID, job.UserID, err)
			_ = discord.Log(ctx, fmt.Sprintf("[Discounted Wishlist Email Ack Failed]\nJob: %s\nUser: %s\nEmail: %s\nAck error: %v", job.ID, job.UserID, maskEmail(job.User.Email), err))
			continue
		}
		log.Printf("discounted wishlist email sent: job_id=%s user_id=%s", job.ID, job.UserID)
		_ = discord.Log(ctx, fmt.Sprintf("[Discounted Wishlist Email Sent]\nJob: %s\nUser: %s\nEmail: %s", job.ID, job.UserID, maskEmail(job.User.Email)))
	}
}

type discountedWishlistRetryQueue interface {
	Enqueue(ctx context.Context, job domain.DiscountedWishlistJob) error
	EnqueueTo(ctx context.Context, name string, job domain.DiscountedWishlistJob) error
}

func handleFailedDiscountedWishlistJob(ctx context.Context, redisQueue discountedWishlistRetryQueue, auditLogger *audit.Logger, auditInfo audit.MessageInfo, job domain.DiscountedWishlistJob, processErr error, cfg senderConfig) (failedJobResult, error) {
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

func runWishlistBackInSender(ctx context.Context, q *queue.WishlistBackInRedisQueue, processor *worker.WishlistBackInProcessor, discord notify.DiscordLogger, auditLogger *audit.Logger, cfg senderConfig) {
	if cfg.rateLimitPerMinute <= 0 {
		cfg.rateLimitPerMinute = 100
	}
	if recovered, err := q.RecoverProcessing(ctx); err != nil {
		log.Printf("wishlist back in redis processing recovery failed: %v", err)
	} else if recovered > 0 {
		log.Printf("wishlist back in redis processing recovery requeued %d job(s)", recovered)
	}
	interval := time.Minute / time.Duration(cfg.rateLimitPerMinute)
	if interval <= 0 {
		interval = time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		processingJob, err := q.Dequeue(ctx, 5*time.Second)
		if errors.Is(err, redis.Nil) {
			continue
		}
		if err != nil {
			log.Printf("wishlist back in redis dequeue failed: %v", err)
			continue
		}
		job := processingJob.Job
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		processResult, err := processor.Process(ctx, job)
		auditInfo := auditInfoFromWishlistBackInJob(job, processResult)
		if err != nil {
			auditInfo.FailureReason = err.Error()
			if auditErr := auditLogger.MarkFailed(ctx, auditInfo); auditErr != nil {
				log.Printf("wishlist back in email audit failed update failed: job_id=%s err=%v", job.ID, auditErr)
			}
			if ackErr := q.Ack(ctx, processingJob.Payload); ackErr != nil {
				log.Printf("wishlist back in email failed but ack failed: job_id=%s err=%v", job.ID, ackErr)
			}
			_ = discord.Log(ctx, fmt.Sprintf("[Wishlist Back In Email Failed]\nJob: %s\nUser: %s\nEmail: %s\nError: %v", job.ID, job.UserID, maskEmail(job.User.Email), err))
			continue
		}
		if err := auditLogger.MarkSent(ctx, auditInfo); err != nil {
			log.Printf("wishlist back in email audit sent update failed: job_id=%s err=%v", job.ID, err)
		}
		if err := q.Ack(ctx, processingJob.Payload); err != nil {
			log.Printf("wishlist back in email sent but ack failed: job_id=%s err=%v", job.ID, err)
			continue
		}
		log.Printf("wishlist back in email sent: job_id=%s user_id=%s", job.ID, job.UserID)
	}
}

func auditInfoFromWinbackJob(job domain.WinbackJob, result worker.ProcessResult) audit.MessageInfo {
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

func runWinbackSender(ctx context.Context, q *queue.WinbackRedisQueue, processor *worker.WinbackProcessor, discord notify.DiscordLogger, auditLogger *audit.Logger, cfg senderConfig) {
	if cfg.rateLimitPerMinute <= 0 {
		cfg.rateLimitPerMinute = 100
	}
	if cfg.maxAttempts <= 0 {
		cfg.maxAttempts = 3
	}
	if recovered, err := q.RecoverProcessing(ctx); err != nil {
		log.Printf("winback redis processing recovery failed: %v", err)
	} else if recovered > 0 {
		log.Printf("winback redis processing recovery requeued %d job(s)", recovered)
	}
	interval := time.Minute / time.Duration(cfg.rateLimitPerMinute)
	if interval <= 0 {
		interval = time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		processingJob, err := q.Dequeue(ctx, 5*time.Second)
		if errors.Is(err, redis.Nil) {
			continue
		}
		if err != nil {
			log.Printf("winback redis dequeue failed: %v", err)
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
		auditInfo := auditInfoFromWinbackJob(job, processResult)
		if err != nil {
			log.Printf("winback email failed: job_id=%s user_id=%s err=%v", job.ID, job.UserID, err)
			_ = discord.Log(ctx, fmt.Sprintf("[Winback Email Failed]\nJob: %s\nUser: %s\nEmail: %s\nError: %v", job.ID, job.UserID, maskEmail(job.User.Email), err))
			if ackErr := q.Ack(ctx, processingJob.Payload); ackErr != nil {
				log.Printf("winback email failed but ack failed: job_id=%s err=%v ack_err=%v", job.ID, err, ackErr)
			}
			if err := auditLogger.MarkFailed(ctx, auditInfo); err != nil {
				log.Printf("winback email audit failed update failed: job_id=%s err=%v", job.ID, err)
			}
			continue
		}

		if err := auditLogger.MarkSent(ctx, auditInfo); err != nil {
			log.Printf("winback email audit sent update failed: job_id=%s user_id=%s err=%v", job.ID, job.UserID, err)
		}
		if err := q.Ack(ctx, processingJob.Payload); err != nil {
			log.Printf("winback email sent but ack failed: job_id=%s user_id=%s err=%v", job.ID, job.UserID, err)
			_ = discord.Log(ctx, fmt.Sprintf("[Winback Email Ack Failed]\nJob: %s\nUser: %s\nEmail: %s\nAck error: %v", job.ID, job.UserID, maskEmail(job.User.Email), err))
			continue
		}
		log.Printf("winback email sent: job_id=%s user_id=%s", job.ID, job.UserID)
		_ = discord.Log(ctx, fmt.Sprintf("[Winback Email Sent]\nJob: %s\nUser: %s\nEmail: %s", job.ID, job.UserID, maskEmail(job.User.Email)))
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
