package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/kyou-id/makoto/internal/audit"
	"github.com/kyou-id/makoto/internal/notify"
)

// campaignRun accumulates one drain of one campaign's queue: everything Makoto did
// between the first job it picked up and the moment the queue went quiet.
//
// The counts of what Makoto *did* (sent, failed, skipped) are held here rather than
// queried back, because Makoto is the authority on them. What the provider did
// afterwards — delivered, bounced, opened — is not knowable here and is read from
// the audit rows the webhooks folded into, keyed on the job ids collected below.
type campaignRun struct {
	feature string // audit feature, e.g. "winback"
	title   string // display name, e.g. "Winback"
	queue   string

	mu           sync.Mutex
	startedAt    time.Time
	lastActivity time.Time
	jobIDs       []string
	sent         int
	failed       int
	deadLetter   int
	skipped      int
	requeued     int
}

func newCampaignRun(feature, title, queue string) *campaignRun {
	return &campaignRun{feature: feature, title: title, queue: queue}
}

// record notes one finished job. outcome is one of sent/failed/dead-letter/skipped/
// requeued.
func (r *campaignRun) record(jobID, outcome string, now time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.startedAt.IsZero() {
		r.startedAt = now
	}
	r.lastActivity = now
	r.jobIDs = append(r.jobIDs, jobID)

	switch outcome {
	case "sent":
		r.sent++
	case "failed":
		r.failed++
	case "dead-letter":
		r.deadLetter++
	case "skipped":
		r.skipped++
	case "requeued":
		r.requeued++
	}
}

// due reports whether the queue has been quiet long enough to summarise. The settle
// delay is not politeness: Kirim.email's delivered/bounced webhooks arrive seconds
// to minutes AFTER the send, so a summary posted the instant the queue drains would
// report zero delivered and zero bounced every single time — and with per-email
// alerts off, a bounce would then be invisible everywhere.
func (r *campaignRun) due(settle time.Duration, now time.Time) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return !r.startedAt.IsZero() && now.Sub(r.lastActivity) >= settle
}

// runBatch is a finished run, lifted out of the tracker. It deliberately carries no
// mutex: it is passed around by value, and copying a lock is a bug go vet catches.
type runBatch struct {
	feature    string
	title      string
	queue      string
	startedAt  time.Time
	jobIDs     []string
	sent       int
	failed     int
	deadLetter int
	skipped    int
	requeued   int
}

// take empties the run and returns what it held, so the next batch starts clean.
func (r *campaignRun) take() runBatch {
	r.mu.Lock()
	defer r.mu.Unlock()

	taken := runBatch{
		feature:    r.feature,
		title:      r.title,
		queue:      r.queue,
		startedAt:  r.startedAt,
		jobIDs:     r.jobIDs,
		sent:       r.sent,
		failed:     r.failed,
		deadLetter: r.deadLetter,
		skipped:    r.skipped,
		requeued:   r.requeued,
	}

	r.startedAt = time.Time{}
	r.lastActivity = time.Time{}
	r.jobIDs = nil
	r.sent, r.failed, r.deadLetter, r.skipped, r.requeued = 0, 0, 0, 0, 0

	return taken
}

// summarize posts the run to Discord once its queue has been quiet for `settle`.
// It is called from the sender loop on every idle tick, and does nothing until the
// run is actually due.
func summarize(ctx context.Context, run *campaignRun, auditLogger *audit.Logger, discord notify.DiscordLogger, settle time.Duration, now time.Time) {
	if !run.due(settle, now) {
		return
	}
	batch := run.take()

	outcomes, err := auditLogger.DeliveryOutcomes(ctx, batch.jobIDs)
	if err != nil {
		// Report anyway: Makoto's own counts are the important half, and a summary
		// missing its delivery figures beats no summary at all.
		log.Printf("%s summary: delivery outcomes unavailable: %v", batch.feature, err)
	}

	log.Printf("%s run summary: processed=%d sent=%d failed=%d dead=%d skipped=%d delivered=%d bounced=%d",
		batch.feature, len(batch.jobIDs), batch.sent, batch.failed, batch.deadLetter, batch.skipped,
		outcomes.Delivered, outcomes.Bounced)

	if err := discord.LogEmbed(ctx, summaryEmbed(batch, outcomes, now)); err != nil {
		log.Printf("%s summary: discord post failed: %v", batch.feature, err)
	}
}

func summaryEmbed(batch runBatch, outcomes audit.DeliveryOutcomes, now time.Time) notify.Embed {
	processed := len(batch.jobIDs)
	elapsed := now.Sub(batch.startedAt).Round(time.Second)

	// Red when nothing landed, amber when some of it did not, green otherwise. The
	// colour is the whole point of the embed: it is what you read from across a room.
	color := notify.ColorSuccess
	switch {
	case batch.sent == 0 && processed > 0:
		color = notify.ColorDanger
	case batch.failed > 0 || batch.deadLetter > 0 || outcomes.Bounced > 0:
		color = notify.ColorWarning
	}

	fields := []notify.Field{
		{Name: "📤 Diproses", Value: fmt.Sprintf("**%d** job", processed), Inline: true},
		{Name: "✅ Terkirim", Value: fmt.Sprintf("**%d**", batch.sent), Inline: true},
		{Name: "📬 Sampai", Value: deliveredValue(outcomes.Delivered, batch.sent), Inline: true},
	}

	// Only show the bad news when there is bad news. A wall of zeroes trains people
	// to stop reading the embed, and then the one that matters slips past.
	if problems := problemField(batch, outcomes); problems != "" {
		fields = append(fields, notify.Field{Name: "⚠️ Bermasalah", Value: problems, Inline: false})
	}
	if engagement := engagementField(outcomes); engagement != "" {
		fields = append(fields, notify.Field{Name: "👀 Interaksi", Value: engagement, Inline: false})
	}

	fields = append(fields, notify.Field{
		Name:   "🕐 Durasi",
		Value:  fmt.Sprintf("%s · antrean `%s`", elapsed, batch.queue),
		Inline: false,
	})

	return notify.Embed{
		Title:     "📊 " + batch.title + " · Ringkasan Pengiriman",
		Color:     color,
		Fields:    fields,
		Footer:    "Kyou Email System · angka pengiriman dari Kirim.email",
		Timestamp: now,
	}
}

// deliveredValue shows delivered against sent, because "1.150 delivered" means
// nothing without knowing whether 1.150 or 12.000 went out.
func deliveredValue(delivered, sent int) string {
	if sent == 0 {
		return fmt.Sprintf("**%d**", delivered)
	}
	return fmt.Sprintf("**%d** (%d%%)", delivered, delivered*100/sent)
}

func problemField(batch runBatch, outcomes audit.DeliveryOutcomes) string {
	var lines []string
	if outcomes.Bounced > 0 {
		lines = append(lines, fmt.Sprintf("⚠️ %d bounced", outcomes.Bounced))
	}
	if batch.failed > 0 {
		lines = append(lines, fmt.Sprintf("❌ %d gagal kirim", batch.failed))
	}
	if batch.deadLetter > 0 {
		lines = append(lines, fmt.Sprintf("💀 %d dead-letter", batch.deadLetter))
	}
	if batch.skipped > 0 {
		lines = append(lines, fmt.Sprintf("⏭️ %d dilewati", batch.skipped))
	}
	if batch.requeued > 0 {
		lines = append(lines, fmt.Sprintf("🔁 %d dicoba ulang", batch.requeued))
	}
	return strings.Join(lines, "\n")
}

func engagementField(outcomes audit.DeliveryOutcomes) string {
	if outcomes.Opened == 0 && outcomes.Clicked == 0 {
		return ""
	}
	return fmt.Sprintf("👀 %d dibuka · 🖱️ %d diklik", outcomes.Opened, outcomes.Clicked)
}
