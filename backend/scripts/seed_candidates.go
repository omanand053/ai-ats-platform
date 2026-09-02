package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type authResponse struct {
	Data struct {
		AccessToken string `json:"access_token"`
	} `json:"data"`
}

type apiResponse struct {
	Data json.RawMessage `json:"data"`
	Msg  string          `json:"message"`
}

type roleTemplate struct {
	Designation string
	Skills      []string
	Summary     string
	Paragraphs  []string
}

type candidateSeed struct {
	Name             string   `json:"name"`
	Email            string   `json:"email"`
	ExperienceYears  int      `json:"experience_years"`
	CurrentCompany   string   `json:"current_company"`
	CurrentDesignation string `json:"current_designation"`
	Location         string   `json:"location"`
	Skills           []string `json:"skills"`
	Source           string   `json:"source"`
	Status           string   `json:"status"`
	ResumeSummary    string   `json:"resume_summary"`
	ResumeText       string   `json:"resume_text"`
}

var firstNames = []string{
	"Aarav", "Ananya", "Arjun", "Ishita", "Rohan", "Meera", "Vihaan", "Sanya", "Kabir", "Pooja",
	"Nikhil", "Priya", "Rahul", "Neha", "Karan", "Riya", "Siddharth", "Tanya", "Aditya", "Maya",
}

var lastNames = []string{
	"Sharma", "Iyer", "Gupta", "Nair", "Patel", "Reddy", "Kulkarni", "Malhotra", "Verma", "Chopra",
}

var companies = []string{
	"TechNova", "BluePeak Systems", "NexaWorks", "CloudHarbor", "ByteForge", "Quantix Labs", "PulseStack", "Northstar Digital", "VertexSoft", "OrbitGrid",
}

var locations = []string{
	"Bengaluru", "Hyderabad", "Pune", "Chennai", "Gurugram", "Noida", "Mumbai", "Delhi", "Kochi", "Ahmedabad",
}

var roles = []roleTemplate{
	{
		Designation: "Backend Engineer",
		Skills:      []string{"Go", "PostgreSQL", "Redis", "Docker", "REST API"},
		Summary:     "Backend engineer focused on scalable APIs, data consistency, and service reliability.",
		Paragraphs: []string{
			"Built and maintained Go services that handled authentication, billing, and internal workflows for a B2B SaaS platform serving thousands of daily requests. Partnered with product and QA teams to define API contracts, stabilize releases, and reduce regressions through better test coverage and observability.",
			"Designed PostgreSQL schemas, tuned indexes, and rewrote expensive queries that improved p95 latency and reduced database load during peak usage. Introduced Redis-backed caching for frequently read endpoints and cache invalidation rules that kept responses fresh without overloading the primary database.",
			"Worked closely with DevOps to containerize services with Docker, streamline CI checks, and ship deployments through automated pipelines. Participated in incident reviews, documented operational playbooks, and helped on-call teams troubleshoot issues using logs, traces, and metrics.",
			"Frequently collaborated with frontend engineers to simplify payloads, version APIs safely, and make integrations predictable for external clients. Comfortable working in distributed systems, multi-service architectures, and environments where reliability, security, and maintainability matter equally.",
		},
	},
	{
		Designation: "Full Stack Engineer",
		Skills:      []string{"TypeScript", "React", "Go", "PostgreSQL", "Docker"},
		Summary:     "Full stack engineer who builds product features end-to-end across web, API, and infrastructure layers.",
		Paragraphs: []string{
			"Delivered customer-facing workflows in React and TypeScript while keeping the matching Go APIs clean, consistent, and easy to extend. Worked on forms, dashboards, tables, and role-based access flows used by operations and customer success teams.",
			"Created reusable UI patterns, improved loading states, and reduced client-side bugs by tightening validation and server-side contracts. On the backend, shaped PostgreSQL queries, designed API endpoints, and used Docker-based local environments to keep development reproducible.",
			"Partnered with designers and product managers to iterate quickly on feature scope, then translated those requirements into technical plans and release milestones. Tracked performance bottlenecks, fixed edge-case issues, and wrote tests for the pieces most likely to regress.",
			"Comfortable switching between frontend polish and backend architecture. Recent work included audit trails, notification preferences, searchable activity logs, and improving the developer experience for teams shipping multiple releases each week.",
		},
	},
	{
		Designation: "Platform Engineer",
		Skills:      []string{"Go", "Kubernetes", "Docker", "Terraform", "PostgreSQL"},
		Summary:     "Platform engineer building internal tooling, deployment pipelines, and production reliability improvements.",
		Paragraphs: []string{
			"Supported product teams by building internal Go services and platform tooling that simplified deployments, environment management, and observability. Focused on reducing toil by standardizing service templates, secrets handling, and release procedures.",
			"Improved container workflows with Docker and Kubernetes, helping teams move from manual deployments to automated rollouts with safer rollback paths. Collaborated on infrastructure-as-code changes and made sure configuration drift was detected early.",
			"Investigated performance, reliability, and cost tradeoffs across services that depended on PostgreSQL, caches, queues, and external APIs. Created runbooks and dashboards that made incidents easier to diagnose and reduced mean time to recovery.",
			"Works well with application engineers who need practical platform solutions instead of abstract architecture. Strong in balancing developer experience with the realities of security reviews, production stability, and operational support.",
		},
	},
	{
		Designation: "DevOps Engineer",
		Skills:      []string{"Docker", "Kubernetes", "CI/CD", "Linux", "Go"},
		Summary:     "DevOps engineer focused on deployments, automation, reliability, and release safety.",
		Paragraphs: []string{
			"Owned CI/CD pipelines and release automation for multiple services, ensuring builds were reproducible and deployments were predictable. Improved branch checks, artifact handling, and environment promotion rules to reduce release risk.",
			"Managed containerized workloads, maintained Kubernetes manifests, and collaborated with developers to create safer defaults for health checks, resource requests, and rollout strategies. Helped teams understand how infrastructure decisions affected application behavior.",
			"Used scripting and Go utilities to automate operational tasks such as config validation, deployment status checks, and environment provisioning. Also spent time improving alert quality so on-call engineers were paged for meaningful issues instead of noise.",
			"Comfortable operating across incident response, change management, and long-term platform hardening. Brings a practical mindset to tuning systems for reliability while keeping delivery velocity high.",
		},
	},
	{
		Designation: "Data Engineer",
		Skills:      []string{"Python", "SQL", "PostgreSQL", "Airflow", "Docker"},
		Summary:     "Data engineer who builds pipelines, models clean datasets, and keeps analytics systems dependable.",
		Paragraphs: []string{
			"Built batch and incremental data pipelines that moved operational data into analytics-ready structures for reporting and forecasting teams. Worked closely with analysts to make sure transformation logic matched business definitions and downstream consumers could trust the data.",
			"Designed SQL transformations, optimized warehouse queries, and improved data quality checks for records arriving from APIs and internal services. Used Dockerized local tooling to keep pipeline development, testing, and debugging consistent across the team.",
			"Partnered with backend engineers to understand source-of-truth systems, schema changes, and event timing issues that affected reporting accuracy. Added validation steps to catch broken upstream data before it reached dashboards or ML workflows.",
			"Comfortable balancing pipeline reliability, schema evolution, and documentation so both technical and non-technical stakeholders can understand what the data means. Enjoys work that connects product behavior with measurable business outcomes.",
		},
	},
	{
		Designation: "Frontend Engineer",
		Skills:      []string{"TypeScript", "React", "Next.js", "CSS", "REST API"},
		Summary:     "Frontend engineer focused on responsive interfaces, accessibility, and reliable client experiences.",
		Paragraphs: []string{
			"Built polished interfaces in React and TypeScript for dashboards, administration panels, and workflow-heavy product areas. Paid close attention to accessibility, keyboard navigation, and visual hierarchy so users could complete tasks quickly and confidently.",
			"Worked with backend teams to integrate REST APIs, handle loading and error states, and reduce UI complexity by simplifying response shapes. Improved the consistency of form validation, empty states, and table interactions across the product.",
			"Contributed to design system work by standardizing reusable components and improving the developer experience for teams that needed to ship features rapidly. Used performance profiling and careful rendering patterns to keep pages responsive under real-world data loads.",
			"Brings a strong product mindset to frontend work and is comfortable collaborating with design, engineering, and QA. Enjoys turning ambiguous requirements into interfaces that feel clear, maintainable, and pleasant to use.",
		},
	},
	{
		Designation: "SRE Engineer",
		Skills:      []string{"Go", "Linux", "Kubernetes", "Prometheus", "Docker"},
		Summary:     "Site reliability engineer building observability, resilience, and incident response improvements.",
		Paragraphs: []string{
			"Helped production teams improve service reliability by building monitoring, alerting, and automation around critical systems. Focused on catching degradation early and giving responders the context they needed to act quickly.",
			"Worked across Kubernetes, Linux, and container tooling to standardize operational patterns and reduce deployment-related failures. Partnered with application teams to create sensible health checks, rollout strategies, and recovery procedures.",
			"Used Go for internal tooling that automated repetitive tasks, summarized incidents, and exposed useful diagnostics for engineers. Also helped refine runbooks so on-call responders could follow a consistent path from alert to resolution.",
			"Brings a calm, methodical approach to production issues and prefers durable fixes over temporary workarounds. Values service health, observability, and engineering practices that keep systems predictable under stress.",
		},
	},
	{
		Designation: "Security Engineer",
		Skills:      []string{"Go", "Application Security", "IAM", "Docker", "PostgreSQL"},
		Summary:     "Security engineer focused on application hardening, identity, and secure delivery practices.",
		Paragraphs: []string{
			"Worked with product teams to improve authentication, authorization, and sensitive data handling across customer-facing services. Reviewed application behavior for access control gaps, input validation issues, and insecure defaults that could become production risks.",
			"Supported secure delivery by helping engineering teams establish safer deployment practices, secrets management, and environment segmentation. Created practical guidance that balanced security controls with the need to ship quickly and confidently.",
			"Investigated application and infrastructure findings, prioritized remediation work, and followed up on fixes with measurable verification steps. Familiar with secure coding patterns, dependency review, and tracing issues across services and databases.",
			"Enjoys working with developers who want security feedback that is specific, actionable, and easy to apply. Focuses on reducing real-world risk without turning every change into a bottleneck.",
		},
	},
	{
		Designation: "Machine Learning Engineer",
		Skills:      []string{"Python", "SQL", "Docker", "ML Ops", "PostgreSQL"},
		Summary:     "Machine learning engineer working on data pipelines, model support systems, and production-ready ML workflows.",
		Paragraphs: []string{
			"Built feature pipelines and model-support services that made experimentation and deployment easier for data science teams. Coordinated with product stakeholders to understand what signals mattered and how model outputs would be consumed by downstream systems.",
			"Created tooling around training data preparation, experiment tracking, and reproducible environments using Docker and SQL-based workflows. Helped reduce the gap between notebook experimentation and production usage by documenting assumptions and constraints clearly.",
			"Collaborated with backend engineers to integrate model outputs into existing APIs, making sure latency, failure handling, and fallback logic were considered early. Cared about observability, versioning, and the safety of model rollouts.",
			"Enjoys bridging product, data, and engineering concerns in environments where model quality has to meet real operational standards. Focused on making machine learning systems reliable, explainable, and useful.",
		},
	},
	{
		Designation: "QA Automation Engineer",
		Skills:      []string{"Go", "Playwright", "API Testing", "Docker", "SQL"},
		Summary:     "QA automation engineer who builds confidence in releases through reliable test coverage and tooling.",
		Paragraphs: []string{
			"Designed automated test coverage for customer workflows, backend APIs, and regression-prone product areas. Partnered with developers and product owners to identify the highest-value scenarios and ensure releases had meaningful safety nets.",
			"Created maintainable test suites and supporting utilities that reduced flakiness and made failures easier to diagnose. Used API-level checks, database assertions, and browser automation together to validate end-to-end behavior.",
			"Worked closely with engineering teams to improve test data management, CI stability, and feedback speed in pull requests. Helped define quality gates that balanced coverage with practical delivery timelines.",
			"Comfortable in environments where quality is shared across teams, not isolated in one function. Brings an analytical approach to release risk, root-cause analysis, and building tests that matter.",
		},
	},
}

func main() {
	baseURL := envOrDefault("BASE_URL", "http://localhost:8000")
	seedEmail := envOrDefault("EMAIL", "candidates-test@example.com")
	seedPassword := envOrDefault("PASSWORD", "Password@123")
	companyName := envOrDefault("COMPANY_NAME", "Candidates Test")
	companySlug := envOrDefault("COMPANY_SLUG", fmt.Sprintf("candidates-test-%d", time.Now().Unix()))

	start := time.Now()

	token, err := getJWT(baseURL, seedEmail, seedPassword, companyName, companySlug)
	if err != nil {
		fmt.Fprintf(os.Stderr, "auth failed: %v\n", err)
		os.Exit(1)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	totalCreated := 0
	totalFailed := 0

	for i := 0; i < 100; i++ {
		profile := buildCandidate(i)
		if err := createCandidate(client, baseURL, token, profile); err != nil {
			totalFailed++
			fmt.Printf("[%03d/100] failed: %s (%s) - %v\n", i+1, profile.Name, profile.Email, err)
		} else {
			totalCreated++
		}

		if (i+1)%10 == 0 {
			fmt.Printf("Progress: %d/100 processed, created=%d failed=%d\n", i+1, totalCreated, totalFailed)
		}

		time.Sleep(150 * time.Millisecond)
	}

	elapsed := time.Since(start).Round(time.Second)
	fmt.Printf("Success summary: total created=%d total failed=%d elapsed=%s\n", totalCreated, totalFailed, elapsed)
}

func getJWT(baseURL, email, password, companyName, companySlug string) (string, error) {
	token, status, err := loginToken(baseURL, email, password)
	if err != nil {
		return "", err
	}
	if token != "" {
		return token, nil
	}

	if status == http.StatusUnauthorized {
		signupBody := map[string]string{
			"company_name": companyName,
			"company_slug": companySlug,
			"email":        email,
			"password":     password,
			"first_name":   "Test",
			"last_name":    "User",
		}
		if _, _, err := doRequest(http.MethodPost, baseURL+"/api/v1/auth/signup", signupBody, ""); err != nil {
			return "", err
		}

		token, _, err = loginToken(baseURL, email, password)
		if err != nil {
			return "", err
		}
		if token != "" {
			return token, nil
		}
	}

	return "", fmt.Errorf("no access token returned from login")
}

func loginToken(baseURL, email, password string) (string, int, error) {
	loginBody := map[string]string{
		"email":    email,
		"password": password,
	}
	body, status, err := doRequest(http.MethodPost, baseURL+"/api/v1/auth/login", loginBody, "")
	if err != nil {
		return "", 0, err
	}

	var loginResp authResponse
	if status == http.StatusOK {
		if err := json.Unmarshal(body, &loginResp); err != nil {
			return "", status, err
		}
		return loginResp.Data.AccessToken, status, nil
	}

	return "", status, nil
}

func buildCandidate(index int) candidateSeed {
	role := roles[index%len(roles)]
	first := firstNames[index/len(lastNames)%len(firstNames)]
	last := lastNames[index%len(lastNames)]
	name := fmt.Sprintf("%s %s", first, last)
	email := fmt.Sprintf("%s.%s.%02d@example.com", strings.ToLower(first), strings.ToLower(last), index+1)
	company := companies[index%len(companies)]
	location := locations[index%len(locations)]
	experience := 2 + (index % 9)

	resumeText := buildResumeText(role, name, company, location, experience, index)

	return candidateSeed{
		Name:               name,
		Email:              email,
		ExperienceYears:    experience,
		CurrentCompany:     company,
		CurrentDesignation: role.Designation,
		Location:           location,
		Skills:             role.Skills,
		Source:             "Seed Script",
		Status:             candidateStatus(index),
		ResumeSummary:      role.Summary,
		ResumeText:         resumeText,
	}
}

func candidateStatus(index int) string {
	statuses := []string{"applied", "screening", "shortlisted", "interview"}
	return statuses[index%len(statuses)]
}

func buildResumeText(role roleTemplate, name, company, location string, experience, index int) string {
	paragraphs := make([]string, 0, len(role.Paragraphs)+2)
	opening := fmt.Sprintf("%s is a %s with %d years of experience based in %s. The current role at %s has involved ownership of core product workflows, collaboration with cross-functional teams, and steady delivery of maintainable software in production environments.", name, role.Designation, experience, location, company)
	paragraphs = append(paragraphs, opening)
	paragraphs = append(paragraphs, role.Paragraphs...)
	closing := fmt.Sprintf("In recent work, %s has focused on writing clear technical documentation, improving operational visibility, and shipping changes that are easy for other engineers to support. The candidate is comfortable working with modern product teams, reviewing tradeoffs carefully, and contributing to systems that need reliability, security, and predictable iteration. Index %d reflects one profile in the generated seed set, but the experience pattern and project framing remain realistic for the selected role.", name, index+1)
	paragraphs = append(paragraphs, closing)
	return strings.Join(paragraphs, "\n\n")
}

func createCandidate(client *http.Client, baseURL, token string, candidate candidateSeed) error {
	body, err := json.Marshal(candidate)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/v1/candidates", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	return nil
}

func doRequest(method, url string, payload any, token string) ([]byte, int, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, 0, err
	}

	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	responseBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return responseBytes, resp.StatusCode, nil
	}

	return responseBytes, resp.StatusCode, nil
}

func envOrDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}