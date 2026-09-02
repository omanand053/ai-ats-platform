package assistant

import (
	"context"
	"fmt"
	"strings"

	"ai-ats-platform/backend/internal/llm"
	"ai-ats-platform/backend/internal/repository"
	"ai-ats-platform/backend/internal/service"

	"github.com/google/uuid"
)

const (
	ragSystemPrompt = `You are an AI recruiting assistant.
Answer ONLY using the retrieved document context below.
If the context is insufficient, say you could not find enough supporting information.
Never invent candidates, skills, scores, or facts not present in the context.
Be concise and professional.`

	generalSystemPrompt = `You are an AI assistant for an Applicant Tracking System.
Answer general recruiting, career, and technology questions clearly in 5–12 lines.
Do not invent company-specific ATS data. If the user needs live ATS numbers, suggest they ask about candidates, jobs, or the hiring funnel.`

	attachedResumeSystemPrompt = `You are an AI recruiting assistant. Answer using ONLY the provided resume text.
If the resume lacks the information, say so clearly. Be concise.`
)

// Router is the intent-based assistant entrypoint (no LangChain).
// Flow: DetectIntent → ATS_DATA | RAG | GENERAL.
type Router struct {
	llm        llm.Provider
	ats        *ATSDataService
	embeddings *service.EmbeddingService
	embRepo    *repository.EmbeddingRepository
	resumes    *repository.ResumeRepository
	ragTopK    int
}

// RouterDeps wires existing services into the assistant router.
type RouterDeps struct {
	LLM        llm.Provider
	Jobs       *service.JobService
	Candidates *service.CandidateService
	Enterprise *service.EnterpriseService
	Embeddings *service.EmbeddingService
	EmbRepo    *repository.EmbeddingRepository
	Resumes    *repository.ResumeRepository
	RAGTopK    int
}

func NewRouter(deps RouterDeps) *Router {
	topK := deps.RAGTopK
	if topK < 1 {
		topK = 5
	}
	return &Router{
		llm:        deps.LLM,
		ats:        NewATSDataService(deps.Jobs, deps.Candidates, deps.Enterprise),
		embeddings: deps.Embeddings,
		embRepo:    deps.EmbRepo,
		resumes:    deps.Resumes,
		ragTopK:    topK,
	}
}

// AskRequest is the chat input.
type AskRequest struct {
	CompanyID uuid.UUID
	Query     string
	ResumeID  string // optional: attached resume forces grounded RAG on that document
}

// Ask routes the query and returns a structured Response.
func (r *Router) Ask(ctx context.Context, req AskRequest) (*Response, error) {
	query := strings.TrimSpace(req.Query)
	if query == "" {
		return nil, fmt.Errorf("query is required")
	}

	intent := DetectIntent(query)
	if strings.TrimSpace(req.ResumeID) != "" {
		intent = IntentRAG
	}

	var resp *Response
	var err error
	switch intent {
	case IntentATSData:
		resp, err = r.runATS(ctx, req.CompanyID, query)
	case IntentRAG:
		resp, err = r.runRAG(ctx, req.CompanyID, query, req.ResumeID)
	default:
		intent = IntentGeneral
		resp, err = r.runGeneral(ctx, query)
	}
	if err != nil {
		return nil, err
	}
	resp.Intent = intent
	if resp.Provider == "" && r.llm != nil {
		resp.Provider = r.llm.Name()
		resp.Model = r.llm.Model()
	}
	return resp, nil
}

func (r *Router) runATS(ctx context.Context, companyID uuid.UUID, query string) (*Response, error) {
	result, err := r.ats.Query(ctx, companyID, query)
	if err != nil {
		return nil, err
	}
	confidence := 95.0
	if result.Empty {
		confidence = 100.0
	}
	return &Response{
		Answer:           result.Answer,
		Confidence:       confidence,
		Source:           SourceDatabase,
		Intent:           IntentATSData,
		SuggestedActions: result.Actions,
		Data:             result.Data,
		SourceDocuments:  []SourceDoc{},
		RetrievedChunks:  0,
	}, nil
}

func (r *Router) runRAG(ctx context.Context, companyID uuid.UUID, query, resumeIDStr string) (*Response, error) {
	if resumeIDStr != "" {
		return r.answerAttachedResume(ctx, companyID, resumeIDStr, query)
	}

	retriever := NewDocumentRetriever(companyID, r.embeddings, r.embRepo, r.resumes, r.ragTopK)
	docs, err := retriever.GetRelevantDocuments(ctx, query)
	if err != nil {
		return &Response{
			Answer:           InsufficientRAGContext,
			Confidence:       0,
			Source:           SourceDocuments,
			Intent:           IntentRAG,
			SuggestedActions: []string{"Attach a resume", "Ask an ATS data question"},
			SourceDocuments:  []SourceDoc{},
			RetrievedChunks:  0,
		}, nil
	}
	confidence := RAGConfidence(docs)
	sources := SourceDocsFrom(docs)
	if len(docs) == 0 || confidence < ragConfidenceFloor {
		return &Response{
			Answer:           InsufficientRAGContext,
			Confidence:       confidence,
			Source:           SourceDocuments,
			Intent:           IntentRAG,
			SuggestedActions: []string{"Attach a resume", "Ask an ATS data question"},
			SourceDocuments:  sources,
			RetrievedChunks:  len(docs),
		}, nil
	}
	return r.answerFromDocs(ctx, docs, sources, confidence, query, []string{
		"Compare with JD", "Generate Interview", "Review strengths",
	})
}

func (r *Router) answerAttachedResume(ctx context.Context, companyID uuid.UUID, resumeIDStr, query string) (*Response, error) {
	if r.resumes == nil {
		return nil, fmt.Errorf("resume service unavailable")
	}
	resumeID, err := uuid.Parse(strings.TrimSpace(resumeIDStr))
	if err != nil {
		return nil, fmt.Errorf("invalid resume_id")
	}
	resume, err := r.resumes.GetByID(ctx, resumeID, companyID)
	if err != nil || resume == nil {
		return nil, fmt.Errorf("resume not found")
	}
	text := ""
	if resume.ParsedText != nil {
		text = strings.TrimSpace(*resume.ParsedText)
	}
	if text == "" {
		return &Response{
			Answer:           "Resume text is not available yet. Wait for parsing to finish, then ask again.",
			Confidence:       0,
			Source:           SourceDocuments,
			Intent:           IntentRAG,
			SuggestedActions: []string{"Re-attach the resume", "Ask without attachment"},
			SourceDocuments:  []SourceDoc{},
		}, nil
	}
	runes := []rune(text)
	if len(runes) > 12000 {
		text = string(runes[:12000]) + "…"
	}
	doc := Document{
		ID: resume.ID.String(), Content: text, Similarity: 100,
		Metadata: map[string]string{"label": resume.FileName, "type": "resume", "file_name": resume.FileName},
	}
	sources := SourceDocsFrom([]Document{doc})
	if r.llm == nil {
		return &Response{
			Answer: InsufficientRAGContext, Confidence: 0, Source: SourceDocuments, Intent: IntentRAG,
			SuggestedActions: []string{"Configure GEMINI_API_KEY"}, SourceDocuments: sources, RetrievedChunks: 1,
		}, nil
	}
	gen, err := r.llm.Generate(ctx, llm.GenerateRequest{
		SystemPrompt: attachedResumeSystemPrompt,
		UserPrompt:   "Resume:\n" + text + "\n\nQuestion: " + query,
		Question:     query,
		Context:      text,
	})
	if err != nil {
		return nil, err
	}
	return &Response{
		Answer:           strings.TrimSpace(gen.Answer),
		Confidence:       90,
		Source:           SourceDocuments,
		Intent:           IntentRAG,
		SuggestedActions: []string{"Extract skills", "Generate interview questions", "Summarize strengths"},
		Provider:         gen.Provider,
		Model:            gen.Model,
		SourceDocuments:  sources,
		RetrievedChunks:  1,
	}, nil
}

func (r *Router) answerFromDocs(
	ctx context.Context,
	docs []Document,
	sources []SourceDoc,
	confidence float64,
	query string,
	actions []string,
) (*Response, error) {
	contextBlock := FormatRAGContext(docs)
	if r.llm == nil {
		return &Response{
			Answer: InsufficientRAGContext, Confidence: confidence, Source: SourceDocuments,
			Intent: IntentRAG, SuggestedActions: []string{"Configure GEMINI_API_KEY"},
			SourceDocuments: sources, RetrievedChunks: len(docs),
		}, nil
	}
	gen, err := r.llm.Generate(ctx, llm.GenerateRequest{
		SystemPrompt: ragSystemPrompt,
		UserPrompt: fmt.Sprintf(
			"Retrieved context (%d chunks):\n%s\n\nQuestion: %s\n\nProvide a grounded answer with brief citations to document labels when possible.",
			len(docs), contextBlock, query,
		),
		Question: query,
		Context:  contextBlock,
	})
	if err != nil {
		return nil, err
	}
	return &Response{
		Answer:           strings.TrimSpace(gen.Answer),
		Confidence:       confidence,
		Source:           SourceDocuments,
		Intent:           IntentRAG,
		SuggestedActions: actions,
		Provider:         gen.Provider,
		Model:            gen.Model,
		SourceDocuments:  sources,
		RetrievedChunks:  len(docs),
	}, nil
}

func (r *Router) runGeneral(ctx context.Context, query string) (*Response, error) {
	if r.llm == nil {
		return &Response{
			Answer:           "General knowledge answers require a configured LLM provider.",
			Confidence:       0,
			Source:           SourceLLM,
			Intent:           IntentGeneral,
			SuggestedActions: []string{"Configure GEMINI_API_KEY", "Ask an ATS data question"},
		}, nil
	}
	gen, err := r.llm.Generate(ctx, llm.GenerateRequest{
		SystemPrompt: generalSystemPrompt,
		UserPrompt:   query,
		Question:     query,
	})
	if err != nil {
		return nil, err
	}
	return &Response{
		Answer:           strings.TrimSpace(gen.Answer),
		Confidence:       70,
		Source:           SourceLLM,
		Intent:           IntentGeneral,
		SuggestedActions: []string{"Ask about your hiring funnel", "Attach a resume", "Find React candidates"},
		Provider:         gen.Provider,
		Model:            gen.Model,
		SourceDocuments:  []SourceDoc{},
	}, nil
}
