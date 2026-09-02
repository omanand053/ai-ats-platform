package service

import (
	"testing"

	"ai-ats-platform/backend/internal/domain"
)

func TestNormalizeCandidateStatusSelected(t *testing.T) {
	got := normalizeCandidateStatus("selected")
	if got != domain.CandidateStatusSelected {
		t.Fatalf("expected %q, got %q", domain.CandidateStatusSelected, got)
	}
}
