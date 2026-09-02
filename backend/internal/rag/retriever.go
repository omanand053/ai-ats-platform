package rag

import (
	"context"
	"fmt"
	"strings"

	"ai-ats-platform/backend/internal/domain"
	"ai-ats-platform/backend/internal/repository"

	"github.com/google/uuid"
)

// SemanticSearcher retrieves Top-K resume↔job matches (implemented by EmbeddingService).
type SemanticSearcher interface {
	SemanticMatchesForJob(
		ctx context.Context,
		companyID, jobID uuid.UUID,
		topK int,
	) (*domain.SemanticMatchResult, error)
}

// RetrievedDoc is one resume snippet retrieved for RAG context.
type RetrievedDoc struct {
	Match          domain.SemanticMatch
	ResumeText     string
	ResumeFileName string
}

// Retriever fetches Top-K semantically relevant resumes for a job via existing semantic search.
type Retriever struct {
	searcher SemanticSearcher
	resumes  *repository.ResumeRepository
}

func NewRetriever(searcher SemanticSearcher, resumes *repository.ResumeRepository) *Retriever {
	return &Retriever{searcher: searcher, resumes: resumes}
}

// Retrieve runs semantic match then loads resume text for only those hits.
func (r *Retriever) Retrieve(
	ctx context.Context,
	companyID, jobID uuid.UUID,
	topK int,
) (*domain.SemanticMatchResult, []RetrievedDoc, error) {
	if r == nil || r.searcher == nil {
		return nil, nil, fmt.Errorf("retriever not configured")
	}
	if topK < 1 {
		topK = 5
	}

	matches, err := r.searcher.SemanticMatchesForJob(ctx, companyID, jobID, topK)
	if err != nil {
		return nil, nil, err
	}
	if matches == nil {
		matches = &domain.SemanticMatchResult{
			JobID:   jobID,
			TopK:    topK,
			Matches: []domain.SemanticMatch{},
			Status:  domain.SemanticStatusNoMatches,
			Message: "No matches",
		}
	}
	if matches.Status != domain.SemanticStatusOK || len(matches.Matches) == 0 {
		return matches, []RetrievedDoc{}, nil
	}

	ids := make([]uuid.UUID, 0, len(matches.Matches))
	for _, m := range matches.Matches {
		if m.ResumeID == uuid.Nil {
			continue
		}
		ids = append(ids, m.ResumeID)
	}
	resumes, err := r.resumes.GetByIDs(ctx, companyID, ids)
	if err != nil {
		return nil, nil, err
	}

	docs := make([]RetrievedDoc, 0, len(matches.Matches))
	for _, m := range matches.Matches {
		text := ""
		fileName := ""
		if resume, ok := resumes[m.ResumeID]; ok {
			if resume.ParsedText != nil {
				text = strings.TrimSpace(*resume.ParsedText)
			}
			fileName = strings.TrimSpace(resume.FileName)
		}
		docs = append(docs, RetrievedDoc{
			Match:          m,
			ResumeText:     text,
			ResumeFileName: fileName,
		})
	}
	return matches, docs, nil
}
