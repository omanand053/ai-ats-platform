package service_test

import (
	"errors"
	"net"
	"testing"

	"ai-ats-platform/backend/internal/llm"
	"ai-ats-platform/backend/internal/rag"
	"ai-ats-platform/backend/internal/service"
)

func TestResumeLabel(t *testing.T) {
	if got := rag.ResumeLabel("Ada", "ada.pdf"); got != "Ada (ada.pdf)" {
		t.Fatalf("got %q", got)
	}
	if got := rag.ResumeLabel("Ada", ""); got != "Resume - Ada" {
		t.Fatalf("got %q", got)
	}
}

func TestClassifyGeminiFallbackReason(t *testing.T) {
	cases := []struct {
		err  error
		want string
	}{
		{errors.New("gemini api key is not configured"), "Missing API key"},
		{
			&llm.APIError{
				Status:   404,
				Body:     `{"error":{"message":"models/gemini-2.0-flash is not found","status":"NOT_FOUND"}}`,
				Model:    "gemini-2.0-flash",
				Endpoint: "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent",
			},
			`configured Gemini model "gemini-2.0-flash" is invalid or unavailable (NOT_FOUND: models/gemini-2.0-flash is not found). Update GEMINI_MODEL in .env to a supported model.`,
		},
		{&net.DNSError{Err: "no such host", Name: "generativelanguage.googleapis.com", IsNotFound: true}, "Network error"},
	}
	for _, tc := range cases {
		got := service.ClassifyGeminiFallbackReasonForTest(tc.err)
		if got != tc.want {
			t.Fatalf("err=%v\ngot  %q\nwant %q", tc.err, got, tc.want)
		}
	}
}
