// Package similarity converts vector distances into recruiter-facing scores.
package similarity

// PercentFromCosineDistance converts pgvector's cosine distance (`<=>`) into a
// 0–100 similarity percentage for display.
//
// pgvector definition:
//
//	cosine_distance = 1 - cosine_similarity
//	cosine_distance ∈ [0, 2]
//	cosine_similarity ∈ [-1, 1]
//
// Therefore:
//
//	cosine_similarity = 1 - cosine_distance
//	percent           = clamp(cosine_similarity, 0, 1) × 100
//
// Negative cosine similarity (distance > 1) is treated as 0% — no positive
// directional match. This does not change ranking; callers must still order by
// raw distance ascending (or similarity descending).
func PercentFromCosineDistance(distance float64) float64 {
	sim := 1 - distance
	if sim <= 0 {
		return 0
	}
	if sim >= 1 {
		return 100
	}
	return sim * 100
}
