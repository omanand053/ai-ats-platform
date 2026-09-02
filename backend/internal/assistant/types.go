package assistant

// Intent classification for the AI Recruiting Assistant.
type Intent string

const (
	IntentATSData Intent = "ATS_DATA"
	IntentRAG     Intent = "RAG"
	IntentGeneral Intent = "GENERAL"
)

// Source identifies where the answer came from.
type Source string

const (
	SourceDatabase  Source = "database"
	SourceDocuments Source = "documents"
	SourceLLM       Source = "general_knowledge"
	SourceNone      Source = "none"
)

// Response is the unified assistant output format.
type Response struct {
	Answer           string         `json:"answer"`
	Confidence       float64        `json:"confidence"`
	Source           Source         `json:"source"`
	Intent           Intent         `json:"intent"`
	SuggestedActions []string       `json:"suggested_actions"`
	Provider         string         `json:"provider,omitempty"`
	Model            string         `json:"model,omitempty"`
	SourceDocuments  []SourceDoc    `json:"source_documents,omitempty"`
	RetrievedChunks  int            `json:"retrieved_context_count,omitempty"`
	Data             map[string]any `json:"data,omitempty"`
}

// SourceDoc is a cited document used in a RAG answer.
type SourceDoc struct {
	ID         string  `json:"id,omitempty"`
	Label      string  `json:"label"`
	Type       string  `json:"type,omitempty"`
	Similarity float64 `json:"similarity,omitempty"`
}

// Document is a retrieved text chunk for RAG.
type Document struct {
	ID         string
	Content    string
	Similarity float64
	Metadata   map[string]string
}

// NoMatchingATSRecords is returned when ATS_DATA finds nothing.
const NoMatchingATSRecords = "No matching records were found in your ATS."

// InsufficientRAGContext is returned when RAG confidence/context is too low.
const InsufficientRAGContext = "I couldn't find enough supporting information in your uploaded documents."

const ragConfidenceFloor = 35.0
const maxResumeChars = 2500
