ALTER TABLE candidates
    ADD COLUMN IF NOT EXISTS overall_score NUMERIC(5,2),
    ADD COLUMN IF NOT EXISTS score_breakdown JSONB,
    ADD COLUMN IF NOT EXISTS matched_skills TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS missing_skills TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS last_scored_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_candidates_overall_score
    ON candidates (company_id, overall_score DESC NULLS LAST);
