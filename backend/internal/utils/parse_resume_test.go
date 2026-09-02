package utils

import (
	"strings"
	"testing"
)

func TestParseResumeTextCoreFields(t *testing.T) {
	text := `Jane Doe
jane.doe@example.com | +1 555 010 1234
Bangalore, India

Summary
Software engineer with 5 years of experience building APIs.

Experience
Senior Software Engineer at Acme Corp
Built recruiting platforms.

Education
B.S. Computer Science, Example University
2015 - 2019

Certifications
AWS Certified Solutions Architect
Google Cloud Professional

Skills
Go, java script, React.js, mongo db, Docker

Projects
AI ATS Platform | Go, React
Recruiter tooling for screening candidates.
`

	parsed := ParseResumeText(text)
	if parsed.FullName != "Jane Doe" {
		t.Fatalf("name=%q", parsed.FullName)
	}
	if parsed.Email != "jane.doe@example.com" {
		t.Fatalf("email=%q", parsed.Email)
	}
	if parsed.ExperienceYears == nil || *parsed.ExperienceYears != 5 {
		t.Fatalf("experience=%v", parsed.ExperienceYears)
	}
	if parsed.CurrentCompany != "Acme Corp" {
		t.Fatalf("company=%q", parsed.CurrentCompany)
	}
	if parsed.CurrentDesignation != "Senior Software Engineer" {
		t.Fatalf("role=%q", parsed.CurrentDesignation)
	}
	if parsed.Location != "Bangalore, India" {
		t.Fatalf("location=%q", parsed.Location)
	}
	if len(parsed.Education) == 0 || parsed.Education[0].School == "" {
		t.Fatalf("education=%v", parsed.Education)
	}
	if len(parsed.Certifications) < 2 {
		t.Fatalf("certifications=%v", parsed.Certifications)
	}
	skillSet := map[string]bool{}
	for _, s := range parsed.Skills {
		skillSet[NormalizeSkillKey(s)] = true
	}
	for _, want := range []string{"javascript", "react", "mongodb", "go", "docker"} {
		if !skillSet[NormalizeSkillKey(want)] {
			t.Fatalf("missing skill %s in %v", want, parsed.Skills)
		}
	}
	if parsed.Summary == "" || !containsAny(parsed.Summary, []string{"Software engineer"}) {
		t.Fatalf("summary=%q", parsed.Summary)
	}
}

func TestNormalizeSkillAliases(t *testing.T) {
	pairs := [][2]string{
		{"java script", "JavaScript"},
		{"JS", "JavaScript"},
		{"React.js", "React"},
		{"node js", "Node.js"},
		{"mongo db", "MongoDB"},
	}
	for _, p := range pairs {
		if CanonicalSkillName(p[0]) != p[1] {
			t.Fatalf("%s => %s, want %s", p[0], CanonicalSkillName(p[0]), p[1])
		}
	}
}

func TestParseResumeBlankWhenMissing(t *testing.T) {
	text := `Alex Only
alex@example.com

Skills
Go, React
`
	parsed := ParseResumeText(text)
	if parsed.CurrentCompany != "" || parsed.CurrentDesignation != "" {
		t.Fatalf("expected blank company/role")
	}
	if parsed.Location != "" {
		t.Fatalf("expected blank location, got %q", parsed.Location)
	}
	if len(parsed.Education) != 0 || len(parsed.Certifications) != 0 {
		t.Fatalf("expected blank education/certs")
	}
}

func TestLabeledFieldsPreferred(t *testing.T) {
	text := `Sam Rivera
sam@example.com
Location: Hyderabad, India
Company: NewCo
Job Title: Senior Developer

Summary
Platform engineer focused on APIs.

Skills
Go, React
`
	parsed := ParseResumeText(text)
	if parsed.Location != "Hyderabad, India" {
		t.Fatalf("location=%q", parsed.Location)
	}
	if parsed.CurrentCompany != "NewCo" {
		t.Fatalf("company=%q", parsed.CurrentCompany)
	}
	if !strings.Contains(parsed.CurrentDesignation, "Senior Developer") {
		t.Fatalf("role=%q", parsed.CurrentDesignation)
	}
}

func TestParseResumeExtractsRoleFromSummaryText(t *testing.T) {
	text := `Swayam Vishwakarma
swayam@example.com

Professional Summary
Full Stack Developer with experience in Python, React and Node.js.

Technical Skills
Python, React, Node.js
`
	parsed := ParseResumeText(text)
	if !strings.Contains(strings.ToLower(parsed.CurrentDesignation), "full stack developer") {
		t.Fatalf("role=%q", parsed.CurrentDesignation)
	}
}
