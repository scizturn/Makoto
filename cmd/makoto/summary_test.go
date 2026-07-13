package main

import (
	"testing"
	"time"

	"github.com/kyou-id/makoto/internal/audit"
	"github.com/kyou-id/makoto/internal/notify"
)

func fieldValue(embed notify.Embed, name string) string {
	for _, field := range embed.Fields {
		if len(field.Name) >= len(name) && field.Name[len(field.Name)-len(name):] == name {
			return field.Value
		}
	}
	return ""
}

// The settle window is the whole reason this is not posted the moment the queue
// empties: Kirim.email's delivered/bounced webhooks arrive AFTER the send.
func TestRunIsNotDueUntilTheQueueHasBeenQuiet(t *testing.T) {
	start := time.Date(2026, 7, 13, 9, 0, 0, 0, time.UTC)
	run := newCampaignRun("winback", "Winback", "winback_email_jobs")

	if run.due(10*time.Minute, start) {
		t.Fatal("an empty run must never be due — it would post a summary of nothing")
	}

	run.record("winback-2026-07-13-user-1", "sent", start)

	if run.due(10*time.Minute, start.Add(9*time.Minute)) {
		t.Fatal("not due before the settle window: the delivery webhooks have not landed yet")
	}
	if !run.due(10*time.Minute, start.Add(10*time.Minute)) {
		t.Fatal("due once the queue has been quiet for the settle window")
	}
}

func TestTakeResetsTheRunSoTheNextBatchStartsClean(t *testing.T) {
	now := time.Now()
	run := newCampaignRun("winback", "Winback", "winback_email_jobs")
	run.record("job-1", "sent", now)
	run.record("job-2", "failed", now)

	batch := run.take()
	if batch.sent != 1 || batch.failed != 1 || len(batch.jobIDs) != 2 {
		t.Fatalf("unexpected batch: %#v", batch)
	}
	if run.due(0, now) {
		t.Fatal("a taken run must be empty, or the next summary double-counts it")
	}
}

func TestSummaryEmbedReportsWhatHappened(t *testing.T) {
	start := time.Date(2026, 7, 13, 3, 0, 0, 0, time.UTC)
	batch := runBatch{
		title: "Winback", queue: "winback_email_jobs", startedAt: start,
		jobIDs: make([]string, 1240), sent: 1198, failed: 30, skipped: 12,
	}
	outcomes := audit.DeliveryOutcomes{Delivered: 1150, Bounced: 12, Opened: 300, Clicked: 40}

	embed := summaryEmbed(batch, outcomes, start.Add(20*time.Minute))

	if embed.Title != "📊 Winback" {
		t.Fatalf("unexpected title %q", embed.Title)
	}
	// Bounces and failures mean the run needs a second look, so it must not be green.
	if embed.Color != notify.ColorWarning {
		t.Fatalf("expected an amber embed, got %#x", embed.Color)
	}
	if got := fieldValue(embed, "Diproses"); got != "**1240** job" {
		t.Fatalf("processed: %q", got)
	}
	// Delivered is meaningless without what it is out of.
	if got := fieldValue(embed, "Sampai"); got != "**1150** (95%)" {
		t.Fatalf("delivered: %q", got)
	}
	if got := fieldValue(embed, "Bermasalah"); got == "" {
		t.Fatalf("a run with 12 bounces and 30 failures must say so")
	}
}

// A clean run must not carry a wall of zeroes: people stop reading those, and then
// the summary that does matter slips past.
func TestACleanRunHidesTheProblemFields(t *testing.T) {
	batch := runBatch{title: "Birthday", jobIDs: make([]string, 20), sent: 20, startedAt: time.Now()}
	outcomes := audit.DeliveryOutcomes{Delivered: 20}

	embed := summaryEmbed(batch, outcomes, time.Now())

	if embed.Color != notify.ColorSuccess {
		t.Fatalf("a clean run should be green, got %#x", embed.Color)
	}
	if got := fieldValue(embed, "Bermasalah"); got != "" {
		t.Fatalf("expected no problem field on a clean run, got %q", got)
	}
}

// Everything failing is not a warning, it is an emergency.
func TestARunWhereNothingSentIsRed(t *testing.T) {
	batch := runBatch{title: "Winback", jobIDs: make([]string, 40), failed: 40, startedAt: time.Now()}

	embed := summaryEmbed(batch, audit.DeliveryOutcomes{}, time.Now())

	if embed.Color != notify.ColorDanger {
		t.Fatalf("expected red when nothing sent, got %#x", embed.Color)
	}
}
