package domain

import (
	"time"

	"github.com/google/uuid"
)

const (
	EmbeddingEntityResume    = "resume"
	EmbeddingEntityCandidate = "candidate"
	EmbeddingEntityJob       = "job"
)

// Embedding stores a vector representation of resume/candidate/job text.
type Embedding struct {
	ID               uuid.UUID `json:"id"`
	CompanyID        uuid.UUID `json:"company_id"`
	EntityType       string    `json:"entity_type"`
	EntityID         uuid.UUID `json:"entity_id"`
	ContentHash      string    `json:"content_hash"`
	Embedding        []float32 `json:"-"`
	EmbeddingModel   string    `json:"embedding_model"`
	EmbeddingVersion string    `json:"embedding_version"`
	EmbeddedAt       time.Time `json:"embedded_at"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// SemanticMatchStatuses for GET /jobs/:id/semantic-matches.
const (
	SemanticStatusOK                   = "ok"
	SemanticStatusJobEmbeddingMissing  = "job_embedding_missing"
	SemanticStatusNoResumeEmbeddings   = "no_resume_embeddings"
	SemanticStatusNoMatches            = "no_matches"
)

// SemanticMatch is one resume↔job evaluation for the Job Workspace.
// Compatibility: similarity_score and overall_score (rule-based eligibility) remain.
// Recruiters should prefer ai_match_score as the ranking signal.
// Eligibility never blocks ranking; low_eligibility is a soft warning badge only.
type SemanticMatch struct {
	CandidateID          uuid.UUID          `json:"candidate_id"`
	CandidateName        string             `json:"candidate_name"`
	ResumeID             uuid.UUID          `json:"resume_id"`
	SimilarityScore      float64            `json:"similarity_score"` // semantic similarity 0–100
	OverallScore         *float64           `json:"overall_score,omitempty"` // rule-based eligibility (legacy)
	AIMatchScore         float64            `json:"ai_match_score"` // Overall AI Match 0–100
	Confidence           string             `json:"confidence"`     // high | medium | low
	Strengths            []string           `json:"strengths"`
	MissingSkills        []string           `json:"missing_skills"`
	MatchedSkills        []string           `json:"matched_skills"`
	WhyShortlisted       string             `json:"why_shortlisted"`
	WhyNotShortlisted    string             `json:"why_not_shortlisted,omitempty"`
	EligibilityScore     *float64           `json:"eligibility_score,omitempty"` // same as overall_score
	EligibilityBreakdown *FitScoreBreakdown `json:"eligibility_breakdown,omitempty"`
	AIMatchBreakdown     *AIMatchBreakdown  `json:"ai_match_breakdown,omitempty"`
	LowEligibility       bool               `json:"low_eligibility"`
}

// AIMatchBreakdown shows weighted dimension scores (each 0–100) for Overall AI Match.
type AIMatchBreakdown struct {
	Semantic   float64 `json:"semantic"`
	Skills     float64 `json:"skills"`
	Experience float64 `json:"experience"`
	Education  float64 `json:"education"`
	Projects   float64 `json:"projects"`
}

// NotShortlistedCandidate explains why an applicant was excluded from ranking.
type NotShortlistedCandidate struct {
	CandidateID       uuid.UUID `json:"candidate_id"`
	CandidateName     string    `json:"candidate_name"`
	Reason            string    `json:"reason"`
	EligibilityScore  *float64  `json:"eligibility_score,omitempty"`
	WhyNotShortlisted string    `json:"why_not_shortlisted"`
}

// SemanticMatchResult is the API payload for semantic candidate search.
type SemanticMatchResult struct {
	Status         string                    `json:"status"`
	Message        string                    `json:"message"`
	JobID          uuid.UUID                 `json:"job_id"`
	TopK           int                       `json:"top_k"`
	Matches        []SemanticMatch           `json:"matches"`
	NotShortlisted []NotShortlistedCandidate `json:"not_shortlisted,omitempty"`
	Weights        *AIMatchWeightInfo        `json:"weights,omitempty"`
}

// AIMatchWeightInfo exposes active Overall AI Match weights to the UI.
type AIMatchWeightInfo struct {
	Semantic   float64 `json:"semantic"`
	Skills     float64 `json:"skills"`
	Experience float64 `json:"experience"`
	Education  float64 `json:"education"`
	Projects   float64 `json:"projects"`
}

// AIAssistantRequest is the recruiter question / copilot payload.
type AIAssistantRequest struct {
	Type         string      `json:"type,omitempty"` // qa|summary|interview|recommendation|insights|jd_optimizer|email|compare
	Question     string      `json:"question"`
	TopK         int         `json:"top_k,omitempty"`
	CandidateID  *uuid.UUID  `json:"candidate_id,omitempty"`
	CandidateIDs []uuid.UUID `json:"candidate_ids,omitempty"` // compare (max 5)
	EmailKind    string      `json:"email_kind,omitempty"`    // interview_invite|offer|rejection|follow_up
	Difficulty   string      `json:"difficulty,omitempty"`    // easy|medium|hard
}

// AIAssistantReferencedCandidate is a candidate cited from retrieval.
type AIAssistantReferencedCandidate struct {
	CandidateID     uuid.UUID `json:"candidate_id"`
	CandidateName   string    `json:"candidate_name"`
	ResumeID        uuid.UUID `json:"resume_id"`
	ResumeFileName  string    `json:"resume_file_name,omitempty"`
	ResumeLabel     string    `json:"resume_label"`
	SimilarityScore float64   `json:"similarity_score"`
	OverallScore    *float64  `json:"overall_score,omitempty"`
}

// AIAssistantResponse is the RAG / copilot answer payload.
type AIAssistantResponse struct {
	Answer               string                           `json:"answer"`
	AIAvailable          bool                             `json:"ai_available"`
	ProviderUsed         string                           `json:"provider_used"`
	Message              string                           `json:"message,omitempty"`
	FallbackReason       string                           `json:"fallback_reason,omitempty"`
	ConfidenceScore      *float64                         `json:"confidence_score,omitempty"`
	ReferencedCandidates []AIAssistantReferencedCandidate `json:"referenced_candidates"`
	SemanticMatchesUsed  []SemanticMatch                  `json:"semantic_matches_used"`
	Provider             string                           `json:"provider,omitempty"`
	Model                string                           `json:"model,omitempty"`
	RetrievalStatus      string                           `json:"retrieval_status,omitempty"`
	RetrievalMessage     string                           `json:"retrieval_message,omitempty"`
	Type                 string                           `json:"type,omitempty"`
	Cached               bool                             `json:"cached,omitempty"`
	Structured           map[string]any                   `json:"structured,omitempty"`
	Explainability       *AIExplainability                `json:"explainability,omitempty"`
}

// AIExplainability accompanies every copilot recommendation.
type AIExplainability struct {
	Reason               string   `json:"reason"`
	Evidence             []string `json:"evidence"`
	Confidence           string   `json:"confidence"`
	RelevantResumeSections []string `json:"relevant_resume_sections"`
}
