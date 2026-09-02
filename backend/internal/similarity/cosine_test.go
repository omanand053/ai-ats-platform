package similarity_test

import (
	"math"
	"testing"

	"ai-ats-platform/backend/internal/similarity"
)

func TestPercentFromCosineDistance(t *testing.T) {
	cases := []struct {
		name     string
		distance float64
		want     float64
	}{
		{"identical", 0, 100},
		{"orthogonal", 1, 0},
		{"opposite", 2, 0},
		{"strong_match", 0.2, 80},
		{"weak_local_hash", 0.996, 0.4},
		{"sample_db", 0.9624371528625488, (1 - 0.9624371528625488) * 100},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := similarity.PercentFromCosineDistance(tc.distance)
			if math.Abs(got-tc.want) > 1e-9 {
				t.Fatalf("distance=%v: got %v want %v", tc.distance, got, tc.want)
			}
		})
	}
}
