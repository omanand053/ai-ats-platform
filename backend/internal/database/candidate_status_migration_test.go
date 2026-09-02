package database

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCandidateStatusMigrationIncludesWorkflowStages(t *testing.T) {
	path := filepath.Join("..", "..", "migrations", "000009_extend_candidates.up.sql")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read migration file: %v", err)
	}

	text := string(content)
	for _, status := range []string{"recruiter_shortlisted", "ai_shortlisted", "selected"} {
		if !strings.Contains(text, status) {
			t.Fatalf("expected migration to include status %q in %s", status, path)
		}
	}
}
