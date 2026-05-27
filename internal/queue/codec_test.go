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
		WishlistItems: []domain.WishlistItem{{
			ID:           "wish-1",
			Name:         "Figure",
			ImageURL:     "https://kyoucdn.id/items/wish.jpg.webp",
			Price:        850000,
			Status:       "ready",
			Manufacturer: "Vocaloid",
			SeriesName:   "Zenless Zone Zero",
		}},
		FYPItems: []domain.FYPItem{{
			ID:           "fyp-1",
			Name:         "Chara",
			Kind:         "character",
			ImageURL:     "https://kyoucdn.id/items/fyp.jpg.webp",
			Price:        150000,
			Status:       "PO",
			Manufacturer: "Good Smile Company",
			SeriesName:   "Honkai: Star Rail",
		}},
		Attempt: 1,
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
	if got.WishlistItems[0].ImageURL != "https://kyoucdn.id/items/wish.jpg.webp" || got.FYPItems[0].ImageURL != "https://kyoucdn.id/items/fyp.jpg.webp" {
		t.Fatalf("expected image urls to round trip, got %#v", got)
	}
	if got.WishlistItems[0].Price != 850000 || got.WishlistItems[0].Status != "ready" || got.WishlistItems[0].Manufacturer != "Vocaloid" || got.WishlistItems[0].SeriesName != "Zenless Zone Zero" || got.FYPItems[0].Price != 150000 || got.FYPItems[0].Status != "PO" || got.FYPItems[0].Manufacturer != "Good Smile Company" || got.FYPItems[0].SeriesName != "Honkai: Star Rail" {
		t.Fatalf("expected commerce fields to round trip, got %#v", got)
	}
}
