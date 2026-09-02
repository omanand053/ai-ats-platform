package embedding_test

import (
	"context"
	"testing"

	"ai-ats-platform/backend/internal/embedding"
)

func TestLocalHashProviderDeterministic(t *testing.T) {
	p := embedding.NewLocalHashProvider("local-hash", "v1", 384)
	a, err := p.Embed(context.Background(), "Senior Go engineer PostgreSQL Docker")
	if err != nil {
		t.Fatal(err)
	}
	b, err := p.Embed(context.Background(), "Senior Go engineer PostgreSQL Docker")
	if err != nil {
		t.Fatal(err)
	}
	if len(a) != 384 || len(b) != 384 {
		t.Fatalf("dims: %d %d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("vectors differ at %d", i)
		}
	}
	c, err := p.Embed(context.Background(), "Completely different product marketing text")
	if err != nil {
		t.Fatal(err)
	}
	same := true
	for i := range a {
		if a[i] != c[i] {
			same = false
			break
		}
	}
	if same {
		t.Fatal("expected different text to produce different vector")
	}
}

func TestContentHashStable(t *testing.T) {
	h1 := embedding.ContentHash("  Hello   World ", "m", "v1")
	h2 := embedding.ContentHash("hello world", "m", "v1")
	if h1 != h2 {
		t.Fatalf("expected normalized hash match: %s vs %s", h1, h2)
	}
	h3 := embedding.ContentHash("hello world", "m", "v2")
	if h1 == h3 {
		t.Fatal("version change should change hash")
	}
}
