package assistant

import "testing"

func TestDetectIntentATS(t *testing.T) {
	cases := []string{
		"Find React candidates",
		"Show jobs",
		"Hiring funnel",
		"How many candidates",
		"Dashboard overview",
		"Highest experience candidates",
	}
	for _, c := range cases {
		if got := DetectIntent(c); got != IntentATSData {
			t.Fatalf("%q => %s, want ATS_DATA", c, got)
		}
	}
}

func TestDetectIntentRAG(t *testing.T) {
	cases := []string{
		"Analyze resume for strengths",
		"Summarize the uploaded resume",
		"Extract skills from the resume",
		"Compare resume with JD",
	}
	for _, c := range cases {
		if got := DetectIntent(c); got != IntentRAG {
			t.Fatalf("%q => %s, want RAG", c, got)
		}
	}
}

func TestDetectIntentGeneral(t *testing.T) {
	cases := []string{
		"What is the STAR method?",
		"Explain React hooks",
		"Career advice for junior engineers",
	}
	for _, c := range cases {
		if got := DetectIntent(c); got != IntentGeneral {
			t.Fatalf("%q => %s, want GENERAL", c, got)
		}
	}
}
