package assistant

import (
	"context"
	"fmt"
	"strings"

	"ai-ats-platform/backend/internal/repository"
	"ai-ats-platform/backend/internal/service"

	"github.com/google/uuid"
)

// DocumentRetriever retrieves company resumes via existing embeddings + vector search.
type DocumentRetriever struct {
	companyID  uuid.UUID
	embeddings *service.EmbeddingService
	embRepo    *repository.EmbeddingRepository
	resumes    *repository.ResumeRepository
	topK       int
}

func NewDocumentRetriever(
	companyID uuid.UUID,
	embeddings *service.EmbeddingService,
	embRepo *repository.EmbeddingRepository,
	resumes *repository.ResumeRepository,
	topK int,
) *DocumentRetriever {
	if topK < 1 {
		topK = 5
	}
	return &DocumentRetriever{
		companyID:  companyID,
		embeddings: embeddings,
		embRepo:    embRepo,
		resumes:    resumes,
		topK:       topK,
	}
}

func (r *DocumentRetriever) GetRelevantDocuments(ctx context.Context, query string) ([]Document, error) {
	if r == nil || r.embeddings == nil || r.embRepo == nil || r.resumes == nil {
		return nil, fmt.Errorf("document retriever not configured")
	}
	vec, err := r.embeddings.EmbedText(ctx, query)
	if err != nil {
		return nil, err
	}
	matches, err := r.embRepo.FindSimilarResumesByVector(
		ctx,
		r.companyID,
		vec,
		r.embeddings.Model(),
		r.embeddings.Version(),
		r.topK,
	)
	if err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		return []Document{}, nil
	}

	ids := make([]uuid.UUID, 0, len(matches))
	for _, m := range matches {
		if m.ResumeID != uuid.Nil {
			ids = append(ids, m.ResumeID)
		}
	}
	resumeMap, err := r.resumes.GetByIDs(ctx, r.companyID, ids)
	if err != nil {
		return nil, err
	}

	docs := make([]Document, 0, len(matches))
	for _, m := range matches {
		res := resumeMap[m.ResumeID]
		text := ""
		label := m.CandidateName
		fileName := ""
		if res != nil {
			if res.ParsedText != nil {
				text = strings.TrimSpace(*res.ParsedText)
			}
			fileName = res.FileName
			if fileName != "" {
				label = fmt.Sprintf("%s (%s)", m.CandidateName, fileName)
			}
		}
		if text == "" {
			continue
		}
		if len(text) > maxResumeChars {
			text = text[:maxResumeChars] + "…"
		}
		docs = append(docs, Document{
			ID:         m.ResumeID.String(),
			Content:    text,
			Similarity: m.SimilarityScore,
			Metadata: map[string]string{
				"candidate_id":   m.CandidateID.String(),
				"candidate_name": m.CandidateName,
				"resume_id":      m.ResumeID.String(),
				"file_name":      fileName,
				"label":          label,
				"type":           "resume",
			},
		})
	}
	return docs, nil
}

// RAGConfidence averages similarity of retrieved docs (0–100).
func RAGConfidence(docs []Document) float64 {
	if len(docs) == 0 {
		return 0
	}
	var sum float64
	for _, d := range docs {
		sum += d.Similarity
	}
	return sum / float64(len(docs))
}

// FormatRAGContext builds the prompt context block from documents.
func FormatRAGContext(docs []Document) string {
	var b strings.Builder
	for i, d := range docs {
		label := d.Metadata["label"]
		if label == "" {
			label = fmt.Sprintf("Document %d", i+1)
		}
		fmt.Fprintf(&b, "--- %s (similarity=%.1f) ---\n%s\n\n", label, d.Similarity, d.Content)
	}
	return strings.TrimSpace(b.String())
}

// SourceDocsFrom converts retrieved docs to API source citations.
func SourceDocsFrom(docs []Document) []SourceDoc {
	out := make([]SourceDoc, 0, len(docs))
	for _, d := range docs {
		out = append(out, SourceDoc{
			ID:         d.ID,
			Label:      d.Metadata["label"],
			Type:       d.Metadata["type"],
			Similarity: d.Similarity,
		})
	}
	return out
}
