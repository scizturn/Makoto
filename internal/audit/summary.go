package audit

import (
	"context"
	"strings"
)

// DeliveryOutcomes is what the provider told us happened to a batch of emails,
// after they left Makoto.
type DeliveryOutcomes struct {
	Delivered int
	Opened    int
	Clicked   int
	Bounced   int
}

// jobIDChunk caps how many job ids go into one IN (…) list. A campaign run can be
// thousands of jobs, and a single statement with thousands of placeholders is both
// slow to plan and liable to hit max_allowed_packet.
const jobIDChunk = 500

// DeliveryOutcomes counts the milestones recorded under metadata.delivery for the
// given jobs — the rows the Kirim.email webhooks folded back in.
//
// It counts *emails that reached a milestone*, not events: a user who opened the
// same email five times is one opened, because only the first timestamp is stored.
func (l *Logger) DeliveryOutcomes(ctx context.Context, jobIDs []string) (DeliveryOutcomes, error) {
	var total DeliveryOutcomes
	if l == nil || l.db == nil || len(jobIDs) == 0 {
		return total, nil
	}

	for start := 0; start < len(jobIDs); start += jobIDChunk {
		chunk := jobIDs[start:min(start+jobIDChunk, len(jobIDs))]

		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(chunk)), ",")
		args := make([]any, 0, len(chunk))
		for _, jobID := range chunk {
			args = append(args, jobID)
		}

		var chunkTotal DeliveryOutcomes
		err := l.db.QueryRowContext(ctx, `
SELECT
  COALESCE(SUM(JSON_EXTRACT(metadata, '$.delivery.delivered_at') IS NOT NULL), 0),
  COALESCE(SUM(JSON_EXTRACT(metadata, '$.delivery.opened_at') IS NOT NULL), 0),
  COALESCE(SUM(JSON_EXTRACT(metadata, '$.delivery.clicked_at') IS NOT NULL), 0),
  COALESCE(SUM(JSON_EXTRACT(metadata, '$.delivery.bounced_at') IS NOT NULL), 0)
FROM email_delivery_logs
WHERE job_id IN (`+placeholders+`)`, args...).Scan(
			&chunkTotal.Delivered,
			&chunkTotal.Opened,
			&chunkTotal.Clicked,
			&chunkTotal.Bounced,
		)
		if err != nil {
			return DeliveryOutcomes{}, err
		}

		total.Delivered += chunkTotal.Delivered
		total.Opened += chunkTotal.Opened
		total.Clicked += chunkTotal.Clicked
		total.Bounced += chunkTotal.Bounced
	}

	return total, nil
}
