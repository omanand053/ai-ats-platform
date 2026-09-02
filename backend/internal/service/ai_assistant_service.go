package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"

	"ai-ats-platform/backend/internal/config"
	"ai-ats-platform/backend/internal/domain"
	"ai-ats-platform/backend/internal/llm"
	"ai-ats-platform/backend/internal/rag"
	"ai-ats-platform/backend/internal/repository"

	"github.com/google/uuid"
)

const (
	fallbackReasonMissingKey = "Missing API key"
	fallbackReasonInvalidKey = "Invalid API key"
	fallbackReasonAPIError   = "Gemini API error"
	fallbackReasonNetwork    = "Network error"

	aiUnavailableMessage = "AI response is unavailable because Gemini could not be used. Showing retrieved candidate data only."
)

// AIAssistantService runs the recruiter RAG pipeline for a job.
type AIAssistantService struct {
	jobs        *repository.JobRepository
	candidates  *repository.CandidateRepository
	retriever   *rag.Retriever
	contextB    *rag.ContextBuilder
	geminiKey   string
	geminiModel string
	defaultTopK int
	cacheMu     sync.RWMutex
	cache       map[string]copilotCacheEntry
}

func NewAIAssistantService(
	jobs *repository.JobRepository,
	retriever *rag.Retriever,
	contextB *rag.ContextBuilder,
	geminiAPIKey string,
	geminiModel string,
	defaultTopK int,
) *AIAssistantService {
	if defaultTopK < 1 {
		defaultTopK = 5
	}
	if contextB == nil {
		contextB = rag.NewContextBuilder()
	}
	return &AIAssistantService{
		jobs:        jobs,
		retriever:   retriever,
		contextB:    contextB,
		geminiKey:   strings.TrimSpace(geminiAPIKey),
		geminiModel: config.NormalizeGeminiModel(geminiModel),
		defaultTopK: defaultTopK,
		cache:       map[string]copilotCacheEntry{},
	}
}

// Ask answers a recruiter question using Top-K semantic resume retrieval + Gemini.
// Gemini is always attempted first. On failure, no fake LLM answer is generated;
// retrieved candidate data is still returned.
func (s *AIAssistantService) Ask(
	ctx context.Context,
	companyID, jobID uuid.UUID,
	question string,
	topK int,
) (*domain.AIAssistantResponse, error) {
	question = strings.TrimSpace(question)
	if question == "" {
		return nil, fmt.Errorf("question is required")
	}
	if topK < 1 {
		topK = s.defaultTopK
	}
	if topK > 20 {
		topK = 20
	}

	job, err := s.jobs.GetByID(ctx, companyID, jobID)
	if err != nil {
		if errors.Is(err, repository.ErrJobNotFound) {
			return nil, ErrJobNotFound
		}
		return nil, err
	}

	matchResult, docs, err := s.retriever.Retrieve(ctx, companyID, jobID, topK)
	if err != nil {
		return nil, err
	}
	if matchResult == nil {
		matchResult = &domain.SemanticMatchResult{
			JobID:   jobID,
			TopK:    topK,
			Matches: []domain.SemanticMatch{},
			Status:  domain.SemanticStatusNoMatches,
		}
	}

	fileByResume := make(map[uuid.UUID]string, len(docs))
	for _, doc := range docs {
		fileByResume[doc.Match.ResumeID] = doc.ResumeFileName
	}

	referenced := make([]domain.AIAssistantReferencedCandidate, 0, len(matchResult.Matches))
	for _, m := range matchResult.Matches {
		fileName := fileByResume[m.ResumeID]
		referenced = append(referenced, domain.AIAssistantReferencedCandidate{
			CandidateID:     m.CandidateID,
			CandidateName:   m.CandidateName,
			ResumeID:        m.ResumeID,
			ResumeFileName:  fileName,
			ResumeLabel:     rag.ResumeLabel(m.CandidateName, fileName),
			SimilarityScore: m.SimilarityScore,
			OverallScore:    m.OverallScore,
		})
	}

	resp := &domain.AIAssistantResponse{
		ReferencedCandidates: referenced,
		SemanticMatchesUsed:  matchResult.Matches,
		RetrievalStatus:      matchResult.Status,
		RetrievalMessage:     matchResult.Message,
		ConfidenceScore:      semanticConfidence(matchResult.Matches),
	}
	if matchResult.Matches == nil {
		resp.SemanticMatchesUsed = []domain.SemanticMatch{}
	}
	if referenced == nil {
		resp.ReferencedCandidates = []domain.AIAssistantReferencedCandidate{}
	}

	contextBlock := ""
	if matchResult.Status == domain.SemanticStatusOK && len(docs) > 0 {
		contextBlock = s.contextB.Build(job, docs)
	} else {
		contextBlock = fmt.Sprintf(
			"(insufficient resume context) retrieval_status=%s message=%s",
			matchResult.Status,
			matchResult.Message,
		)
	}

	systemPrompt := strings.TrimSpace(`
You are an AI Recruiter Assistant for an Applicant Tracking System.
Answer the recruiter's question using ONLY the provided JOB and RETRIEVED CANDIDATE RESUMES context.
If the context is insufficient, say so clearly.
Cite candidates by name when relevant.
Do not invent candidates, skills, or experience that are not in the context.
`)
	userPrompt := fmt.Sprintf("Context:\n%s\n\nRecruiter question: %s", contextBlock, question)

	// Always attempt Gemini first — never synthesize a fake mock answer on failure.
	if reason := s.geminiUnavailableReason(); reason != "" {
		log.Printf("🤖 AI assistant Gemini fallback reason=%q job_id=%s", reason, jobID)
		return s.unavailable(resp, reason), nil
	}

	gemini := llm.NewGeminiProvider(s.geminiKey, s.geminiModel)
	log.Printf(
		"🤖 AI assistant trying Gemini first model=%s key_loaded=%t key_status=%s job_id=%s",
		gemini.Model(),
		s.geminiKey != "",
		config.ResolveGeminiAPIKey(s.geminiKey),
		jobID,
	)
	if err := gemini.VerifyModel(ctx); err != nil {
		reason := classifyGeminiFallbackReason(err)
		log.Printf("🤖 AI assistant Gemini model verification failed reason=%q err=%v job_id=%s", reason, err, jobID)
		return s.unavailable(resp, reason), nil
	}
	gen, err := gemini.Generate(ctx, llm.GenerateRequest{
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
		Question:     question,
		Context:      contextBlock,
	})
	if err != nil {
		reason := classifyGeminiFallbackReason(err)
		log.Printf("🤖 AI assistant Gemini fallback reason=%q err=%v job_id=%s", reason, err, jobID)
		return s.unavailable(resp, reason), nil
	}

	resp.AIAvailable = true
	resp.ProviderUsed = "gemini"
	resp.Answer = gen.Answer
	resp.Provider = gen.Provider
	resp.Model = gen.Model
	resp.Message = ""
	resp.FallbackReason = ""
	return resp, nil
}

func (s *AIAssistantService) geminiUnavailableReason() string {
	switch config.ResolveGeminiAPIKey(s.geminiKey) {
	case config.GeminiKeyMissing:
		return fallbackReasonMissingKey
	case config.GeminiKeyInvalid:
		return fallbackReasonInvalidKey
	}
	if strings.TrimSpace(s.geminiModel) == "" {
		return config.ErrGeminiModelNotConfigured{}.Error()
	}
	return ""
}

func (s *AIAssistantService) unavailable(resp *domain.AIAssistantResponse, reason string) *domain.AIAssistantResponse {
	resp.AIAvailable = false
	resp.ProviderUsed = "mock"
	resp.Provider = "mock"
	resp.Model = ""
	resp.Answer = ""
	resp.Message = aiUnavailableMessage
	resp.FallbackReason = reason
	return resp
}

func classifyGeminiFallbackReason(err error) string {
	if err == nil {
		return fallbackReasonAPIError
	}
	if llm.IsModelConfigError(err) {
		return err.Error()
	}
	var apiErr *llm.APIError
	if errors.As(err, &apiErr) {
		return apiErr.FallbackReason()
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "gemini_model") || strings.Contains(msg, "geminimodel") {
		return err.Error()
	}
	if strings.Contains(msg, "not configured") || strings.Contains(msg, "missing") {
		return fallbackReasonMissingKey
	}
	var netErr net.Error
	if errors.As(err, &netErr) ||
		strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "connection") ||
		strings.Contains(msg, "network") ||
		strings.Contains(msg, "dns") ||
		strings.Contains(msg, "temporary failure") ||
		strings.Contains(msg, "no such host") {
		return fallbackReasonNetwork
	}
	if strings.Contains(msg, "401") ||
		strings.Contains(msg, "403") ||
		strings.Contains(msg, "api key") ||
		strings.Contains(msg, "permission") ||
		strings.Contains(msg, "unauthenticated") {
		return fallbackReasonInvalidKey
	}
	// Prefer the concrete error text over a generic label.
	return err.Error()
}

// ClassifyGeminiFallbackReasonForTest exports classification for unit tests.
func ClassifyGeminiFallbackReasonForTest(err error) string {
	return classifyGeminiFallbackReason(err)
}

// semanticConfidence averages top similarity percents into a 0–100 confidence score.
func semanticConfidence(matches []domain.SemanticMatch) *float64 {
	if len(matches) == 0 {
		return nil
	}
	n := len(matches)
	if n > 5 {
		n = 5
	}
	sum := 0.0
	for i := 0; i < n; i++ {
		sum += matches[i].SimilarityScore
	}
	avg := sum / float64(n)
	if avg < 0 {
		avg = 0
	}
	if avg > 100 {
		avg = 100
	}
	return &avg
}
