package audit

import (
	"context"
	"database/sql"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
)

const (
	FeatureBirthdayVoucher    = "birthday_voucher"
	FeatureAnniversaryVoucher = "anniversary_voucher"
	FeatureLeftoverCart          = "leftover_cart"
	FeatureDiscountedWishlist    = "discounted_wishlist"
	ProviderKirimEmail           = "kirim.email"
)

type Logger struct {
	db *sql.DB
}

type MessageInfo struct {
	JobID             string
	Attempt           int
	QueueName         string
	TemplateID        string
	Subject           string
	ActionURL         string
	ProviderMessageID string
	ProviderStatus    int
	ProviderResponse  string
	FailureReason     string
}

func Open(dsn string) (*Logger, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Logger{db: db}, nil
}

func (l *Logger) Close() error {
	if l == nil || l.db == nil {
		return nil
	}
	return l.db.Close()
}

func (l *Logger) MarkSending(ctx context.Context, info MessageInfo) error {
	if l == nil {
		return nil
	}
	_, err := l.db.ExecContext(ctx, `
UPDATE email_delivery_logs
SET
  status = 'sending',
  template_id = COALESCE(NULLIF(?, ''), template_id),
  subject = COALESCE(NULLIF(?, ''), subject),
  action_url = COALESCE(NULLIF(?, ''), action_url),
  sending_at = NOW(),
  updated_at = NOW()
WHERE job_id = ?
  AND attempt = ?`,
		info.TemplateID,
		info.Subject,
		info.ActionURL,
		info.JobID,
		normalizedAttempt(info.Attempt),
	)
	return err
}

func (l *Logger) MarkSent(ctx context.Context, info MessageInfo) error {
	if l == nil {
		return nil
	}
	_, err := l.db.ExecContext(ctx, `
UPDATE email_delivery_logs
SET
  status = 'sent',
  template_id = COALESCE(NULLIF(?, ''), template_id),
  subject = COALESCE(NULLIF(?, ''), subject),
  action_url = COALESCE(NULLIF(?, ''), action_url),
  provider_message_id = NULLIF(?, ''),
  provider_status_code = ?,
  provider_response = LEFT(?, 4096),
  sent_at = NOW(),
  updated_at = NOW()
WHERE job_id = ?
  AND attempt = ?`,
		info.TemplateID,
		info.Subject,
		info.ActionURL,
		info.ProviderMessageID,
		statusValue(info.ProviderStatus),
		info.ProviderResponse,
		info.JobID,
		normalizedAttempt(info.Attempt),
	)
	return err
}

func (l *Logger) MarkFailed(ctx context.Context, info MessageInfo) error {
	if l == nil {
		return nil
	}
	_, err := l.db.ExecContext(ctx, `
UPDATE email_delivery_logs
SET
  status = 'failed',
  failure_reason = ?,
  provider_status_code = ?,
  provider_response = LEFT(?, 4096),
  failed_at = NOW(),
  updated_at = NOW()
WHERE job_id = ?
  AND attempt = ?`,
		info.FailureReason,
		statusValue(info.ProviderStatus),
		info.ProviderResponse,
		info.JobID,
		normalizedAttempt(info.Attempt),
	)
	return err
}

func (l *Logger) InsertRetryQueued(ctx context.Context, jobID string, attempt int) error {
	if l == nil {
		return nil
	}
	_, err := l.db.ExecContext(ctx, `
INSERT INTO email_delivery_logs (
  feature,
  reference_type,
  reference_id,
  job_id,
  queue_name,
  attempt,
  user_id,
  to_email,
  template_id,
  subject,
  action_url,
  metadata,
  provider,
  status,
  queued_at
)
SELECT
  feature,
  reference_type,
  reference_id,
  job_id,
  queue_name,
  attempt + 1,
  user_id,
  to_email,
  template_id,
  subject,
  action_url,
  metadata,
  provider,
  'queued',
  NOW()
FROM email_delivery_logs
WHERE job_id = ?
  AND attempt = ?
ON DUPLICATE KEY UPDATE
  status = 'queued',
  queued_at = COALESCE(queued_at, NOW()),
  updated_at = NOW()`,
		jobID,
		normalizedAttempt(attempt),
	)
	return err
}

func (l *Logger) MarkDeadLetter(ctx context.Context, info MessageInfo) error {
	if l == nil {
		return nil
	}
	_, err := l.db.ExecContext(ctx, `
UPDATE email_delivery_logs
SET
  status = 'dead_letter',
  failure_reason = ?,
  provider_status_code = ?,
  provider_response = LEFT(?, 4096),
  failed_at = NOW(),
  updated_at = NOW()
WHERE job_id = ?
  AND attempt = ?`,
		info.FailureReason,
		statusValue(info.ProviderStatus),
		info.ProviderResponse,
		info.JobID,
		normalizedAttempt(info.Attempt),
	)
	return err
}

func normalizedAttempt(attempt int) int {
	if attempt <= 0 {
		return 1
	}
	return attempt
}

func statusValue(status int) any {
	if status <= 0 {
		return nil
	}
	return status
}

func UserIDValue(userID string) any {
	if userID == "" {
		return nil
	}
	parsed, err := strconv.ParseUint(userID, 10, 64)
	if err != nil {
		return nil
	}
	return parsed
}
