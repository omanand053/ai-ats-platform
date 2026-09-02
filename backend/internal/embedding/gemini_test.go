package embedding_test

import (
	"context"
	"math"
	"os"
	"strings"
	"testing"

	"ai-ats-platform/backend/internal/embedding"
)

func TestGeminiLiveMeaningfulSimilarity(t *testing.T) {
	key := strings.TrimSpace(os.Getenv("GEMINI_API_KEY"))
	if key == "" {
		t.Skip("GEMINI_API_KEY not set; skipping live Gemini embedding verification")
	}

	p := embedding.NewGeminiProvider(key, "gemini-embedding-001", "v1", 384)
	ctx := context.Background()

	goEng, err := p.Embed(ctx, "Senior Go backend engineer with PostgreSQL, Docker, and Kubernetes experience building REST APIs.")
	if err != nil {
		t.Fatal(err)
	}
	goSimilar, err := p.Embed(ctx, "Golang software engineer experienced in Postgres, containers, and microservices APIs.")
	if err != nil {
		t.Fatal(err)
	}
	unrelated, err := p.Embed(ctx, "Wedding photographer specializing in outdoor portrait sessions and film editing.")
	if err != nil {
		t.Fatal(err)
	}

	simRelated := cosineSim(goEng, goSimilar)
	simUnrelated := cosineSim(goEng, unrelated)
	t.Logf("related=%.4f unrelated=%.4f (%.1f%% vs %.1f%%)",
		simRelated, simUnrelated, simRelated*100, simUnrelated*100)

	if simRelated <= simUnrelated {
		t.Fatalf("related pair should outrank unrelated: related=%.4f unrelated=%.4f", simRelated, simUnrelated)
	}
	if simRelated < 0.4 {
		t.Fatalf("expected meaningful related similarity (>=0.4), got %.4f", simRelated)
	}
	if len(goEng) != 384 {
		t.Fatalf("dims=%d", len(goEng))
	}
}

func cosineSim(a, b []float32) float64 {
	var dot, na, nb float64
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		dot += float64(a[i]) * float64(b[i])
		na += float64(a[i]) * float64(a[i])
		nb += float64(b[i]) * float64(b[i])
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}
