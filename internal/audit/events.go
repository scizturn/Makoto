package audit

import (
	"context"
	"time"

	"github.com/kyou-id/makoto/internal/webhook"
)

// milestoneKey maps a Kirim.email event to the metadata key it stamps. Events not
// listed here (queued, send, deferred, temporary_fail) still move last_event — they
// just have no milestone of their own.
//
// `status` is deliberately not touched by any of this: it is the dedup spine Yukari
// reads (status IN ('queued','sending','sent')), and moving a 'sent' row to 'opened'
// would silently stop suppressing duplicate emails — a user would be mailed twice
// for the sin of opening the first one.
var milestoneKey = map[string]string{
	"delivered":      "delivered_at",
	"bounced":        "bounced_at",
	"permanent_fail": "bounced_at",
	"opened":         "opened_at",
	"clicked":        "clicked_at",
	"unsubscribed":   "unsubscribed_at",
}

// RecordDeliveryEvent folds one webhook event into the job's existing audit row —
// no new table, no new column. The delivery history lives under metadata.delivery,
// and message_guid finally fills provider_message_id, which Kirim.email never gave
// us at send time.
//
// Everything written here is a *first-seen timestamp*, never a counter. Kirim.email
// retries a webhook until it gets a 2xx, so the same event arrives more than once;
// re-stamping a timestamp is harmless, incrementing a count would inflate it with
// nothing to correct it. That is the price of having no per-event table: we can say
// "this was opened", not "this was opened five times".
func (l *Logger) RecordDeliveryEvent(ctx context.Context, event webhook.Record) error {
	if l == nil || l.db == nil {
		return nil
	}

	// No job id means the email went out before Makoto started stamping X-Tags (all
	// 22k+ of them) or the tag was lost. There is no row to fold it into. The caller
	// still logs and still announces a bounce to Discord, so it is visible — it just
	// leaves no trace in the database.
	if event.JobID == "" {
		return nil
	}

	occurred := event.OccurredAt.UTC().Format(time.RFC3339)

	// COALESCE on the existing value keeps the FIRST time an email reached a
	// milestone: a user opening the same mail three times must not keep rewriting
	// opened_at to the latest open.
	//
	// The '$.delivery' pair runs first so the object exists before its children are
	// set; JSON_SET applies its pairs left to right.
	statement := `
UPDATE email_delivery_logs
SET
  provider_message_id = COALESCE(NULLIF(provider_message_id, ''), ?),
  metadata = JSON_SET(
    COALESCE(metadata, JSON_OBJECT()),
    '$.delivery', COALESCE(JSON_EXTRACT(metadata, '$.delivery'), JSON_OBJECT()),
    '$.delivery.message_guid', ?,
    '$.delivery.last_event', ?,
    '$.delivery.last_event_at', ?`
	args := []any{event.MessageGUID, event.MessageGUID, event.EventType, occurred}

	if key, ok := milestoneKey[event.EventType]; ok {
		statement += `,
    '$.delivery.` + key + `', COALESCE(JSON_UNQUOTE(JSON_EXTRACT(metadata, '$.delivery.` + key + `')), ?)`
		args = append(args, occurred)
	}

	statement += `
  ),
  updated_at = NOW()
WHERE job_id = ?
ORDER BY attempt DESC
LIMIT 1`
	// A job can have several rows (one per attempt). The event describes the email
	// that actually went out, which is the newest attempt.
	args = append(args, event.JobID)

	_, err := l.db.ExecContext(ctx, statement, args...)
	return err
}
