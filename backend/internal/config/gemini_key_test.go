package config_test

import (
	"testing"

	"ai-ats-platform/backend/internal/config"
)

func TestResolveGeminiAPIKey(t *testing.T) {
	cases := []struct {
		key  string
		want config.GeminiAPIKeyStatus
	}{
		{"", config.GeminiKeyMissing},
		{"   ", config.GeminiKeyMissing},
		{"short", config.GeminiKeyInvalid},
		{"changeme", config.GeminiKeyInvalid},
		{"your-api-key", config.GeminiKeyInvalid},
		{"AIzaSyDummyTestKeyOK01", config.GeminiKeyOK},
	}
	for _, tc := range cases {
		got := config.ResolveGeminiAPIKey(tc.key)
		if got != tc.want {
			t.Fatalf("key %q: got %s want %s", tc.key, got, tc.want)
		}
	}
}
