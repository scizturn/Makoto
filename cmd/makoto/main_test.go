package main

import (
	"context"
	"testing"

	"github.com/kyou-id/makoto/internal/audit"
	"github.com/kyou-id/makoto/internal/domain"
)

func TestHandleFailedJobRequeuesUntilMaxAttempts(t *testing.T) {
	queue := &fakeRetryQueue{}
	job := domain.BirthdayJob{ID: "job-1", Attempt: 1}

	result, err := handleFailedJob(context.Background(), queue, nil, audit.MessageInfo{JobID: job.ID, Attempt: job.Attempt}, job, assertError("send failed"), senderConfig{
		maxAttempts:     3,
		deadLetterQueue: "birthday_email_jobs_dead",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.state != "requeued" || result.attempt != 2 {
		t.Fatalf("unexpected result: %#v", result)
	}
	if len(queue.requeued) != 1 || queue.requeued[0].Attempt != 2 {
		t.Fatalf("expected job requeued with attempt 2, got %#v", queue.requeued)
	}
	if len(queue.deadLetters) != 0 {
		t.Fatalf("expected no dead letters, got %#v", queue.deadLetters)
	}
}

func TestHandleFailedJobMovesToDeadLetterAtMaxAttempts(t *testing.T) {
	queue := &fakeRetryQueue{}
	job := domain.BirthdayJob{ID: "job-1", Attempt: 3}

	result, err := handleFailedJob(context.Background(), queue, nil, audit.MessageInfo{JobID: job.ID, Attempt: job.Attempt}, job, assertError("send failed"), senderConfig{
		maxAttempts:     3,
		deadLetterQueue: "birthday_email_jobs_dead",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.state != "dead-letter" || result.attempt != 3 {
		t.Fatalf("unexpected result: %#v", result)
	}
	if len(queue.requeued) != 0 {
		t.Fatalf("expected no requeued jobs, got %#v", queue.requeued)
	}
	if len(queue.deadLetters) != 1 || queue.deadLetters[0].name != "birthday_email_jobs_dead" || queue.deadLetters[0].job.Attempt != 3 {
		t.Fatalf("expected dead-letter job, got %#v", queue.deadLetters)
	}
}

func TestHandleFailedDiscountedWishlistJobRequeuesUntilMaxAttempts(t *testing.T) {
	queue := &fakeDiscountedWishlistRetryQueue{}
	job := domain.DiscountedWishlistJob{ID: "discounted-job-1", Attempt: 1}

	result, err := handleFailedDiscountedWishlistJob(context.Background(), queue, nil, audit.MessageInfo{JobID: job.ID}, job, assertError("send failed"), senderConfig{
		maxAttempts:     3,
		deadLetterQueue: "discounted_wishlist_email_jobs_dead",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.state != "requeued" || result.attempt != 2 || len(queue.requeued) != 1 || queue.requeued[0].Attempt != 2 {
		t.Fatalf("unexpected retry result=%#v queue=%#v", result, queue)
	}
}

func TestHandleFailedDiscountedWishlistJobMovesToDeadLetter(t *testing.T) {
	queue := &fakeDiscountedWishlistRetryQueue{}
	job := domain.DiscountedWishlistJob{ID: "discounted-job-1", Attempt: 3}

	result, err := handleFailedDiscountedWishlistJob(context.Background(), queue, nil, audit.MessageInfo{JobID: job.ID}, job, assertError("send failed"), senderConfig{
		maxAttempts:     3,
		deadLetterQueue: "discounted_wishlist_email_jobs_dead",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.state != "dead-letter" || len(queue.deadLetters) != 1 || queue.deadLetters[0].name != "discounted_wishlist_email_jobs_dead" {
		t.Fatalf("unexpected dead-letter result=%#v queue=%#v", result, queue)
	}
}

type assertError string

func (e assertError) Error() string {
	return string(e)
}

type fakeRetryQueue struct {
	requeued    []domain.BirthdayJob
	deadLetters []namedJob
}

type namedJob struct {
	name string
	job  domain.BirthdayJob
}

type fakeDiscountedWishlistRetryQueue struct {
	requeued    []domain.DiscountedWishlistJob
	deadLetters []namedDiscountedWishlistJob
}

type namedDiscountedWishlistJob struct {
	name string
	job  domain.DiscountedWishlistJob
}

func (q *fakeDiscountedWishlistRetryQueue) Enqueue(_ context.Context, job domain.DiscountedWishlistJob) error {
	q.requeued = append(q.requeued, job)
	return nil
}

func (q *fakeDiscountedWishlistRetryQueue) EnqueueTo(_ context.Context, name string, job domain.DiscountedWishlistJob) error {
	q.deadLetters = append(q.deadLetters, namedDiscountedWishlistJob{name: name, job: job})
	return nil
}

func (q *fakeRetryQueue) Enqueue(_ context.Context, job domain.BirthdayJob) error {
	q.requeued = append(q.requeued, job)
	return nil
}

func (q *fakeRetryQueue) EnqueueTo(_ context.Context, name string, job domain.BirthdayJob) error {
	q.deadLetters = append(q.deadLetters, namedJob{name: name, job: job})
	return nil
}
