package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// FitScoreWeights are relative weights for each fit dimension (normalized to sum 1).
type FitScoreWeights struct {
	Skills         float64
	Experience     float64
	Education      float64
	Seniority      float64
	Location       float64
	Certifications float64
}

// AIMatchWeights control Overall AI Match (normalized to sum 1).
// Used after eligibility pre-filter + semantic search.
type AIMatchWeights struct {
	Semantic   float64
	Skills     float64
	Experience float64
	Education  float64
	Projects   float64
}

// EmbeddingConfig controls the embedding provider used by EmbeddingService.
type EmbeddingConfig struct {
	Provider     string
	Model        string
	Version      string
	Dimensions   int
	Workers      int
	TopK         int
	GeminiAPIKey string
}

// LLMConfig controls the chat/completion provider used by the AI Recruiter Assistant.
type LLMConfig struct {
	Provider     string
	Model        string // Resolved chat model (from GEMINI_MODEL, else LLM_MODEL)
	OpenAIAPIKey string
	GeminiAPIKey string
	RAGTopK      int
}

// CandidateFilterConfig controls the pre-filter used before semantic search.
type CandidateFilterConfig struct {
	RoleWeight       float64
	SkillsWeight     float64
	ExperienceWeight float64
	MinScore         float64
}

type Config struct {
	DatabaseURL     string
	Port            string
	AppEnv          string
	JWTSecret       string
	JWTExpiry       time.Duration
	FitScoreWeights FitScoreWeights
	AIMatchWeights  AIMatchWeights
	Embedding       EmbeddingConfig
	LLM             LLMConfig
	CandidateFilter CandidateFilterConfig
}

func Load() (*Config, error) {
	loadDotEnv()

	jwtExpiry, err := time.ParseDuration(getEnv("JWT_EXPIRY", "24h"))
	if err != nil {
		return nil, err
	}

	weights := FitScoreWeights{
		Skills:         getEnvFloat("FIT_WEIGHT_SKILLS", 0.35),
		Experience:     getEnvFloat("FIT_WEIGHT_EXPERIENCE", 0.20),
		Education:      getEnvFloat("FIT_WEIGHT_EDUCATION", 0.15),
		Seniority:      getEnvFloat("FIT_WEIGHT_SENIORITY", 0.15),
		Location:       getEnvFloat("FIT_WEIGHT_LOCATION", 0.10),
		Certifications: getEnvFloat("FIT_WEIGHT_CERTIFICATIONS", 0.05),
	}
	weights = normalizeFitWeights(weights)

	aiMatch := AIMatchWeights{
		Semantic:   getEnvFloat("AI_MATCH_WEIGHT_SEMANTIC", 0.40),
		Skills:     getEnvFloat("AI_MATCH_WEIGHT_SKILLS", 0.25),
		Experience: getEnvFloat("AI_MATCH_WEIGHT_EXPERIENCE", 0.15),
		Education:  getEnvFloat("AI_MATCH_WEIGHT_EDUCATION", 0.10),
		Projects:   getEnvFloat("AI_MATCH_WEIGHT_PROJECTS", 0.10),
	}
	aiMatch = normalizeAIMatchWeights(aiMatch)

	candidateFilter := CandidateFilterConfig{
		RoleWeight:       getEnvFloat("CANDIDATE_FILTER_ROLE_WEIGHT", 0.30),
		SkillsWeight:     getEnvFloat("CANDIDATE_FILTER_SKILLS_WEIGHT", 0.50),
		ExperienceWeight: getEnvFloat("CANDIDATE_FILTER_EXPERIENCE_WEIGHT", 0.20),
		MinScore:         getEnvFloat("CANDIDATE_FILTER_MIN_SCORE", 40),
	}

	dims := getEnvInt("EMBEDDING_DIMENSIONS", 384)
	if dims != 384 {
		// DB schema uses vector(384); keep runtime aligned.
		dims = 384
	}

	geminiKey := strings.TrimSpace(getEnv("GEMINI_API_KEY", ""))
	geminiKey = strings.Trim(geminiKey, `"'`)

	// Chat model: GEMINI_MODEL is authoritative; LLM_MODEL kept as backward-compatible alias.
	geminiModel := NormalizeGeminiModel(getEnv("GEMINI_MODEL", ""))
	if geminiModel == "" {
		geminiModel = NormalizeGeminiModel(getEnv("LLM_MODEL", ""))
	}

	return &Config{
		DatabaseURL:     getEnv("DATABASE_URL", "postgres://ats_user:ats_password@localhost:5435/ats_db?sslmode=disable"),
		Port:            getEnv("PORT", "8000"),
		AppEnv:          getEnv("APP_ENV", "development"),
		JWTSecret:       getEnv("JWT_SECRET", "change-me-in-production"),
		JWTExpiry:       jwtExpiry,
		FitScoreWeights: weights,
		AIMatchWeights:  aiMatch,
		Embedding: EmbeddingConfig{
			// Default to Gemini; NewProvider falls back to local without GEMINI_API_KEY.
			Provider:     getEnv("EMBEDDING_PROVIDER", "gemini"),
			Model:        getEnv("EMBEDDING_MODEL", "gemini-embedding-001"),
			Version:      getEnv("EMBEDDING_VERSION", "v1"),
			Dimensions:   dims,
			Workers:      getEnvInt("EMBEDDING_WORKERS", 2),
			TopK:         getEnvInt("SEMANTIC_SEARCH_TOP_K", 200),
			GeminiAPIKey: geminiKey,
		},
		LLM: LLMConfig{
			// Default to Gemini; NewProvider falls back to mock without GEMINI_API_KEY.
			Provider:     getEnv("LLM_PROVIDER", "gemini"),
			Model:        geminiModel,
			OpenAIAPIKey: getEnv("OPENAI_API_KEY", ""),
			GeminiAPIKey: geminiKey,
			RAGTopK:      getEnvInt("RAG_TOP_K", 5),
		},
		CandidateFilter: candidateFilter,
	}, nil
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvFloat(key string, fallback float64) float64 {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil || v < 0 {
		return fallback
	}
	return v
}

func getEnvInt(key string, fallback int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 {
		return fallback
	}
	return v
}

func normalizeFitWeights(w FitScoreWeights) FitScoreWeights {
	sum := w.Skills + w.Experience + w.Education + w.Seniority + w.Location + w.Certifications
	if sum <= 0 {
		return FitScoreWeights{
			Skills: 0.35, Experience: 0.20, Education: 0.15,
			Seniority: 0.15, Location: 0.10, Certifications: 0.05,
		}
	}
	w.Skills /= sum
	w.Experience /= sum
	w.Education /= sum
	w.Seniority /= sum
	w.Location /= sum
	w.Certifications /= sum
	return w
}

func normalizeAIMatchWeights(w AIMatchWeights) AIMatchWeights {
	sum := w.Semantic + w.Skills + w.Experience + w.Education + w.Projects
	if sum <= 0 {
		return AIMatchWeights{
			Semantic: 0.40, Skills: 0.25, Experience: 0.15,
			Education: 0.10, Projects: 0.10,
		}
	}
	w.Semantic /= sum
	w.Skills /= sum
	w.Experience /= sum
	w.Education /= sum
	w.Projects /= sum
	return w
}

// loadDotEnv loads the nearest project .env without letting empty process-level
// placeholders block real values (common when shells export KEY=).
func loadDotEnv() {
	for _, key := range []string{"GEMINI_API_KEY", "OPENAI_API_KEY", "LLM_MODEL", "GEMINI_MODEL"} {
		if strings.TrimSpace(os.Getenv(key)) == "" {
			_ = os.Unsetenv(key)
		}
	}

	seen := map[string]struct{}{}
	candidates := []string{".env", "../.env"}
	if wd, err := os.Getwd(); err == nil {
		dir := wd
		for i := 0; i < 6; i++ {
			candidates = append(candidates, filepath.Join(dir, ".env"))
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	for _, path := range candidates {
		clean := filepath.Clean(path)
		if _, ok := seen[clean]; ok {
			continue
		}
		seen[clean] = struct{}{}
		_ = godotenv.Load(clean)
	}
}
