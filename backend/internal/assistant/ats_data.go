package assistant

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"ai-ats-platform/backend/internal/domain"
	"ai-ats-platform/backend/internal/service"

	"github.com/google/uuid"
)

// ATSDataService answers ATS_DATA intents via service → repository → SQL only.
// Never calls an LLM for facts.
type ATSDataService struct {
	jobs       *service.JobService
	candidates *service.CandidateService
	enterprise *service.EnterpriseService
}

func NewATSDataService(
	jobs *service.JobService,
	candidates *service.CandidateService,
	enterprise *service.EnterpriseService,
) *ATSDataService {
	return &ATSDataService{jobs: jobs, candidates: candidates, enterprise: enterprise}
}

// ATSQueryResult is structured DB output before formatting.
type ATSQueryResult struct {
	Answer   string
	Data     map[string]any
	Actions  []string
	Empty    bool
}

func (s *ATSDataService) Query(ctx context.Context, companyID uuid.UUID, query string) (*ATSQueryResult, error) {
	q := strings.ToLower(strings.TrimSpace(query))

	switch {
	case isFunnelQuery(q):
		return s.funnel(ctx, companyID)
	case isInterviewScheduleQuery(q):
		return s.interviews(ctx, companyID)
	case isAvgScoreQuery(q):
		return s.avgScore(ctx, companyID)
	case isCountQuery(q):
		return s.counts(ctx, companyID, q)
	case isJobsThisMonthQuery(q):
		return s.jobsThisMonth(ctx, companyID)
	case isJobQuery(q):
		return s.listJobs(ctx, companyID, q)
	case isTopExperienceQuery(q):
		return s.topExperience(ctx, companyID, q)
	case isTopScoreQuery(q):
		return s.topScores(ctx, companyID, q)
	case isCandidateQuery(q):
		return s.listCandidates(ctx, companyID, q)
	case isDashboardQuery(q):
		return s.dashboard(ctx, companyID)
	default:
		return s.dashboard(ctx, companyID)
	}
}

func (s *ATSDataService) funnel(ctx context.Context, companyID uuid.UUID) (*ATSQueryResult, error) {
	if s.enterprise == nil {
		return emptyATS("Analytics unavailable."), nil
	}
	overview, err := s.enterprise.Overview(ctx, companyID)
	if err != nil {
		return nil, err
	}
	if overview == nil || (overview.Applicants == 0 && overview.TotalJobs == 0) {
		return emptyATS(NoMatchingATSRecords), nil
	}
	var b strings.Builder
	b.WriteString("Hiring funnel:\n")
	for _, stage := range overview.Funnel {
		fmt.Fprintf(&b, "• %s: %d\n", stage.Name, stage.Count)
	}
	return &ATSQueryResult{
		Answer:  strings.TrimSpace(b.String()),
		Data:    map[string]any{"funnel": overview.Funnel, "by_status": overview.ByStatus},
		Actions: []string{"View candidates by stage", "Open analytics dashboard", "Schedule interviews"},
	}, nil
}

func (s *ATSDataService) interviews(ctx context.Context, companyID uuid.UUID) (*ATSQueryResult, error) {
	if s.enterprise == nil {
		return emptyATS("Interview service unavailable."), nil
	}
	from := time.Now().Add(-24 * time.Hour)
	to := time.Now().Add(30 * 24 * time.Hour)
	list, err := s.enterprise.ListInterviews(ctx, companyID, &from, &to)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return emptyATS(NoMatchingATSRecords), nil
	}
	var b strings.Builder
	b.WriteString("Upcoming / recent interviews:\n")
	limit := min(10, len(list))
	items := make([]map[string]any, 0, limit)
	for i := 0; i < limit; i++ {
		iv := list[i]
		fmt.Fprintf(&b, "• %s — %s (%s) @ %s\n",
			iv.Title, iv.CandidateName, iv.Status, iv.ScheduledAt.Format(time.RFC822))
		items = append(items, map[string]any{
			"id": iv.ID, "title": iv.Title, "candidate": iv.CandidateName,
			"status": iv.Status, "scheduled_at": iv.ScheduledAt,
		})
	}
	return &ATSQueryResult{
		Answer:  strings.TrimSpace(b.String()),
		Data:    map[string]any{"interviews": items},
		Actions: []string{"Open calendar", "Reschedule interview", "Add interview notes"},
	}, nil
}

func (s *ATSDataService) avgScore(ctx context.Context, companyID uuid.UUID) (*ATSQueryResult, error) {
	if s.enterprise == nil {
		return emptyATS("Analytics unavailable."), nil
	}
	overview, err := s.enterprise.Overview(ctx, companyID)
	if err != nil {
		return nil, err
	}
	if overview.AvgAIMatch == nil {
		return emptyATS(NoMatchingATSRecords), nil
	}
	answer := fmt.Sprintf("Average AI score across scored candidates: %.1f", *overview.AvgAIMatch)
	return &ATSQueryResult{
		Answer:  answer,
		Data:    map[string]any{"avg_ai_match": *overview.AvgAIMatch},
		Actions: []string{"Show top scored candidates", "Open AI settings", "Review shortlist"},
	}, nil
}

func (s *ATSDataService) counts(ctx context.Context, companyID uuid.UUID, q string) (*ATSQueryResult, error) {
	if s.enterprise == nil {
		return emptyATS("Analytics unavailable."), nil
	}
	overview, err := s.enterprise.Overview(ctx, companyID)
	if err != nil {
		return nil, err
	}
	answer := fmt.Sprintf(
		"Candidates: %d\nApplications: %d\nInterviews: %d\nOffers: %d\nHired: %d",
		overview.Applicants, overview.Applications, overview.Interviews, overview.Offers, overview.Hired,
	)
	if overview.Applicants == 0 && overview.TotalJobs == 0 {
		return emptyATS(NoMatchingATSRecords), nil
	}
	_ = q
	return &ATSQueryResult{
		Answer: answer,
		Data: map[string]any{
			"candidates": overview.Applicants, "applications": overview.Applications,
			"interviews": overview.Interviews, "offers": overview.Offers, "hired": overview.Hired,
		},
		Actions: []string{"List shortlisted candidates", "View hiring funnel", "Search jobs"},
	}, nil
}

func (s *ATSDataService) dashboard(ctx context.Context, companyID uuid.UUID) (*ATSQueryResult, error) {
	if s.enterprise == nil {
		return emptyATS("Analytics unavailable."), nil
	}
	overview, err := s.enterprise.Overview(ctx, companyID)
	if err != nil {
		return nil, err
	}
	if overview.TotalJobs == 0 && overview.Applicants == 0 {
		return emptyATS(NoMatchingATSRecords), nil
	}
	answer := fmt.Sprintf(
		"Jobs: %d (open: %d, closed: %d)\nApplications: %d\nAI shortlisted: %d\nRecruiter shortlisted: %d\nInterviews: %d\nOffers: %d\nHired: %d",
		overview.TotalJobs, overview.OpenJobs, overview.ClosedJobs,
		overview.Applications, overview.AIShortlisted, overview.RecruiterShortlisted,
		overview.Interviews, overview.Offers, overview.Hired,
	)
	return &ATSQueryResult{
		Answer:  answer,
		Data:    map[string]any{"overview": overview},
		Actions: []string{"View funnel", "List open jobs", "Show shortlisted candidates"},
	}, nil
}

func (s *ATSDataService) jobsThisMonth(ctx context.Context, companyID uuid.UUID) (*ATSQueryResult, error) {
	if s.jobs == nil {
		return emptyATS("Job service unavailable."), nil
	}
	res, err := s.jobs.List(ctx, companyID, 1, 200, "")
	if err != nil {
		return nil, err
	}
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	matched := make([]domain.Job, 0)
	for _, j := range res.Jobs {
		if !j.CreatedAt.Before(start) {
			matched = append(matched, j)
		}
	}
	if len(matched) == 0 {
		return emptyATS(NoMatchingATSRecords), nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Jobs created this month (%d):\n", len(matched))
	items := make([]map[string]any, 0, min(12, len(matched)))
	for i, j := range matched {
		if i >= 12 {
			break
		}
		loc := ""
		if j.Location != nil {
			loc = *j.Location
		}
		fmt.Fprintf(&b, "• %s — %s (%s)\n", j.Title, loc, j.Status)
		items = append(items, map[string]any{"id": j.ID, "title": j.Title, "location": loc, "status": j.Status})
	}
	return &ATSQueryResult{
		Answer:  strings.TrimSpace(b.String()),
		Data:    map[string]any{"jobs": items, "count": len(matched)},
		Actions: []string{"Open job details", "Create a job", "View applicants"},
	}, nil
}

func (s *ATSDataService) listJobs(ctx context.Context, companyID uuid.UUID, q string) (*ATSQueryResult, error) {
	if s.jobs == nil {
		return emptyATS("Job service unavailable."), nil
	}
	status := "open"
	if containsAny(q, []string{"closed", "filled", "expired"}) {
		status = "closed"
	}
	res, err := s.jobs.List(ctx, companyID, 1, 200, status)
	if err != nil {
		return nil, err
	}
	tokens := extractSearchTokens(q, jobStopwords)
	jobs := res.Jobs
	if len(tokens) > 0 {
		jobs = scoreJobs(tokens, res.Jobs)
	}
	if len(jobs) == 0 {
		return emptyATS(NoMatchingATSRecords), nil
	}
	var b strings.Builder
	b.WriteString("Matching jobs:\n")
	items := make([]map[string]any, 0, min(10, len(jobs)))
	for i, j := range jobs {
		if i >= 10 {
			break
		}
		loc := ""
		if j.Location != nil {
			loc = *j.Location
		}
		fmt.Fprintf(&b, "• %s — %s (%s)\n", j.Title, loc, j.Status)
		items = append(items, map[string]any{"id": j.ID, "title": j.Title, "location": loc, "status": j.Status})
	}
	return &ATSQueryResult{
		Answer:  strings.TrimSpace(b.String()),
		Data:    map[string]any{"jobs": items},
		Actions: []string{"Open job", "View semantic matches", "Ask about applicants"},
	}, nil
}

func (s *ATSDataService) listCandidates(ctx context.Context, companyID uuid.UUID, q string) (*ATSQueryResult, error) {
	if s.candidates == nil {
		return emptyATS("Candidate service unavailable."), nil
	}
	status := candidateStatusFromQuery(q)
	search := candidateSearchTerm(q)
	res, err := s.candidates.List(ctx, companyID, 1, 200, status, search, nil, "created_at")
	if err != nil {
		return nil, err
	}
	cands := res.Candidates
	tokens := extractSearchTokens(q, candidateStopwords)
	if search == "" && len(tokens) > 0 {
		cands = scoreCandidates(tokens, res.Candidates)
	}
	if len(cands) == 0 {
		return emptyATS(NoMatchingATSRecords), nil
	}
	var b strings.Builder
	b.WriteString("Matching candidates:\n")
	items := make([]map[string]any, 0, min(10, len(cands)))
	for i, c := range cands {
		if i >= 10 {
			break
		}
		score := ""
		if c.OverallScore != nil {
			score = fmt.Sprintf(" — AI score %.0f", *c.OverallScore)
		}
		fmt.Fprintf(&b, "• %s — %s (%s)%s\n", c.Name, c.Email, c.Status, score)
		items = append(items, map[string]any{
			"id": c.ID, "name": c.Name, "email": c.Email, "status": c.Status, "overall_score": c.OverallScore,
		})
	}
	return &ATSQueryResult{
		Answer:  strings.TrimSpace(b.String()),
		Data:    map[string]any{"candidates": items},
		Actions: []string{"Open candidate profile", "Shortlist candidates", "Compare candidates"},
	}, nil
}

func (s *ATSDataService) topExperience(ctx context.Context, companyID uuid.UUID, q string) (*ATSQueryResult, error) {
	if s.candidates == nil {
		return emptyATS("Candidate service unavailable."), nil
	}
	status := candidateStatusFromQuery(q)
	res, err := s.candidates.List(ctx, companyID, 1, 200, status, "", nil, "created_at")
	if err != nil {
		return nil, err
	}
	cands := append([]domain.Candidate(nil), res.Candidates...)
	sort.SliceStable(cands, func(i, j int) bool {
		yi, yj := 0, 0
		if cands[i].ExperienceYears != nil {
			yi = *cands[i].ExperienceYears
		}
		if cands[j].ExperienceYears != nil {
			yj = *cands[j].ExperienceYears
		}
		return yi > yj
	})
	if len(cands) == 0 {
		return emptyATS(NoMatchingATSRecords), nil
	}
	var b strings.Builder
	b.WriteString("Highest experience candidates:\n")
	items := make([]map[string]any, 0, min(10, len(cands)))
	for i, c := range cands {
		if i >= 10 {
			break
		}
		yrs := 0
		if c.ExperienceYears != nil {
			yrs = *c.ExperienceYears
		}
		fmt.Fprintf(&b, "• %s — %d years (%s)\n", c.Name, yrs, c.Status)
		items = append(items, map[string]any{"id": c.ID, "name": c.Name, "experience_years": yrs, "status": c.Status})
	}
	return &ATSQueryResult{
		Answer:  strings.TrimSpace(b.String()),
		Data:    map[string]any{"candidates": items},
		Actions: []string{"Open candidate", "Filter by skills", "View resume"},
	}, nil
}

func (s *ATSDataService) topScores(ctx context.Context, companyID uuid.UUID, q string) (*ATSQueryResult, error) {
	if s.candidates == nil {
		return emptyATS("Candidate service unavailable."), nil
	}
	status := candidateStatusFromQuery(q)
	res, err := s.candidates.List(ctx, companyID, 1, 200, status, "", nil, "overall_score")
	if err != nil {
		return nil, err
	}
	cands := append([]domain.Candidate(nil), res.Candidates...)
	sort.SliceStable(cands, func(i, j int) bool {
		si, sj := -1.0, -1.0
		if cands[i].OverallScore != nil {
			si = *cands[i].OverallScore
		}
		if cands[j].OverallScore != nil {
			sj = *cands[j].OverallScore
		}
		return si > sj
	})
	scored := make([]domain.Candidate, 0, len(cands))
	for _, c := range cands {
		if c.OverallScore != nil {
			scored = append(scored, c)
		}
	}
	if len(scored) == 0 {
		return emptyATS(NoMatchingATSRecords), nil
	}
	var b strings.Builder
	b.WriteString("Highest AI / resume scores:\n")
	items := make([]map[string]any, 0, min(10, len(scored)))
	for i, c := range scored {
		if i >= 10 {
			break
		}
		fmt.Fprintf(&b, "• %s — score %.1f (%s)\n", c.Name, *c.OverallScore, c.Status)
		items = append(items, map[string]any{"id": c.ID, "name": c.Name, "overall_score": *c.OverallScore, "status": c.Status})
	}
	return &ATSQueryResult{
		Answer:  strings.TrimSpace(b.String()),
		Data:    map[string]any{"candidates": items},
		Actions: []string{"Compare top candidates", "Shortlist", "Open AI Copilot on job"},
	}, nil
}

func emptyATS(msg string) *ATSQueryResult {
	return &ATSQueryResult{
		Answer:  msg,
		Empty:   true,
		Data:    map[string]any{},
		Actions: []string{"Upload resumes", "Create a job", "Ask a general recruiting question"},
	}
}

func isFunnelQuery(q string) bool {
	return containsAny(q, []string{"hiring funnel", "funnel", "pipeline stages"})
}
func isInterviewScheduleQuery(q string) bool {
	return containsAny(q, []string{"interview schedule", "scheduled interview", "upcoming interview", "interviews scheduled"})
}
func isAvgScoreQuery(q string) bool {
	return containsAny(q, []string{"average ai", "avg ai", "average score", "avg score", "average resume score"})
}
func isCountQuery(q string) bool {
	return containsAny(q, []string{
		"how many", "total candidates", "total applicants", "candidate count", "applicant count",
		"number of candidates", "number of applicants", "application stats", "total applications",
	})
}
func isJobsThisMonthQuery(q string) bool {
	return containsAny(q, []string{"jobs created this month", "jobs this month", "created this month"})
}
func isTopExperienceQuery(q string) bool {
	return containsAny(q, []string{"highest experience", "most experience", "top experience", "most years"})
}
func isTopScoreQuery(q string) bool {
	return containsAny(q, []string{"highest score", "top score", "resume score", "highest ai", "top ai score", "most applied job"})
}
func isDashboardQuery(q string) bool {
	return containsAny(q, []string{"dashboard", "overview", "analytics", "summary"})
}
func isJobQuery(q string) bool {
	return containsAny(q, []string{"job", "jobs", "position", "positions", "opening", "openings", "hiring"})
}
func isCandidateQuery(q string) bool {
	return containsAny(q, []string{
		"candidate", "candidates", "applicant", "applicants", "shortlisted", "shortlist",
		"resume", "hired", "interview", "react", "frontend", "backend",
	})
}

var jobStopwords = map[string]struct{}{
	"show": {}, "list": {}, "find": {}, "search": {}, "jobs": {}, "job": {}, "positions": {}, "position": {},
	"available": {}, "open": {}, "openings": {}, "the": {}, "a": {}, "an": {}, "all": {}, "me": {},
}
var candidateStopwords = map[string]struct{}{
	"show": {}, "list": {}, "find": {}, "search": {}, "candidates": {}, "candidate": {},
	"applicants": {}, "applicant": {}, "with": {}, "the": {}, "a": {}, "an": {}, "all": {}, "me": {},
	"shortlisted": {}, "shortlist": {}, "top": {}, "highest": {},
}

func extractSearchTokens(q string, stop map[string]struct{}) []string {
	tokens := tokenize(q)
	out := make([]string, 0, len(tokens))
	for _, t := range tokens {
		if _, ok := stop[t]; ok {
			continue
		}
		out = append(out, t)
	}
	return out
}

func candidateStatusFromQuery(q string) string {
	switch {
	case containsAny(q, []string{"hired", "hires"}):
		return "hired"
	case containsAny(q, []string{"interview"}):
		return "interview"
	case containsAny(q, []string{"ai shortlisted"}):
		return "ai_shortlisted"
	case containsAny(q, []string{"recruiter shortlisted"}):
		return "recruiter_shortlisted"
	case containsAny(q, []string{"shortlisted", "shortlist"}):
		return "shortlisted"
	case containsAny(q, []string{"selected"}):
		return "selected"
	case containsAny(q, []string{"offer"}):
		return "offer"
	case containsAny(q, []string{"rejected"}):
		return "rejected"
	case containsAny(q, []string{"applied"}):
		return "applied"
	}
	return ""
}

func candidateSearchTerm(q string) string {
	tokens := extractSearchTokens(q, candidateStopwords)
	// Drop status words from free-text search
	filtered := make([]string, 0, len(tokens))
	skip := map[string]struct{}{
		"hired": {}, "interview": {}, "shortlisted": {}, "shortlist": {}, "selected": {},
		"offer": {}, "rejected": {}, "applied": {}, "screening": {}, "status": {},
	}
	for _, t := range tokens {
		if _, ok := skip[t]; ok {
			continue
		}
		filtered = append(filtered, t)
	}
	return strings.Join(filtered, " ")
}

func scoreJobs(tokens []string, jobs []domain.Job) []domain.Job {
	type scored struct {
		job   domain.Job
		score int
	}
	var out []scored
	for _, job := range jobs {
		score := 0
		title := strings.ToLower(job.Title)
		dept := ""
		if job.Department != nil {
			dept = strings.ToLower(*job.Department)
		}
		loc := ""
		if job.Location != nil {
			loc = strings.ToLower(*job.Location)
		}
		skills := strings.ToLower(strings.Join(job.RequiredSkills, " "))
		desc := ""
		if job.Description != nil {
			desc = strings.ToLower(*job.Description)
		}
		for _, t := range tokens {
			if strings.Contains(title, t) {
				score += 12
			}
			if strings.Contains(dept, t) {
				score += 8
			}
			if strings.Contains(loc, t) {
				score += 6
			}
			if strings.Contains(skills, t) {
				score += 10
			}
			if strings.Contains(desc, t) {
				score += 4
			}
		}
		if score > 0 {
			out = append(out, scored{job: job, score: score})
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].score > out[j].score })
	jobsOut := make([]domain.Job, 0, len(out))
	for _, s := range out {
		jobsOut = append(jobsOut, s.job)
	}
	return jobsOut
}

func scoreCandidates(tokens []string, candidates []domain.Candidate) []domain.Candidate {
	type scored struct {
		c     domain.Candidate
		score int
	}
	var out []scored
	for _, c := range candidates {
		score := 0
		name := strings.ToLower(c.Name)
		skills := strings.ToLower(strings.Join(c.Skills, " "))
		title := ""
		if c.CurrentDesignation != nil {
			title = strings.ToLower(*c.CurrentDesignation)
		}
		for _, t := range tokens {
			if strings.Contains(name, t) {
				score += 10
			}
			if strings.Contains(skills, t) {
				score += 12
			}
			if strings.Contains(title, t) {
				score += 8
			}
			if strings.Contains(strings.ToLower(c.Status), t) {
				score += 6
			}
		}
		if score > 0 {
			out = append(out, scored{c: c, score: score})
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].score > out[j].score })
	cands := make([]domain.Candidate, 0, len(out))
	for _, s := range out {
		cands = append(cands, s.c)
	}
	return cands
}
