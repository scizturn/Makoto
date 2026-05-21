package queue

import (
	"testing"
	"time"

	"github.com/kyou-id/makoto/internal/domain"
)

func TestEncodeDecodeBirthdayJob(t *testing.T) {
	job := domain.BirthdayJob{
		ID:     "birthday-2026-05-21-user-123",
		UserID: "123",
		Date:   time.Date(2026, 5, 21, 0, 0, 0, 0, time.UTC),
		User: domain.User{
			ID:       "123",
			Name:     "Garvin",
			Email:    "garvin@example.test",
			IsActive: true,
		},
		WishlistItems: []domain.WishlistItem{{ID: "wish-1", Name: "Figure"}},
		FYPItems:      []domain.FYPItem{{ID: "fyp-1", Name: "Chara", Kind: "character"}},
		Attempt:       1,
	}

	payload, err := EncodeBirthdayJob(job)
	if err != nil {
		t.Fatalf("expected encode success, got %v", err)
	}
	got, err := DecodeBirthdayJob(payload)
	if err != nil {
		t.Fatalf("expected decode success, got %v", err)
	}

	if got.ID != job.ID || got.User.Email != job.User.Email || len(got.WishlistItems) != 1 || len(got.FYPItems) != 1 {
		t.Fatalf("unexpected decoded job: %#v", got)
	}
}
