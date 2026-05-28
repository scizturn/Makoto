package audit

import "testing"

func TestNormalizedAttemptDefaultsToOne(t *testing.T) {
	if got := normalizedAttempt(0); got != 1 {
		t.Fatalf("expected attempt 1, got %d", got)
	}
}

func TestStatusValueReturnsNilForUnknownStatus(t *testing.T) {
	if got := statusValue(0); got != nil {
		t.Fatalf("expected nil, got %#v", got)
	}
}
