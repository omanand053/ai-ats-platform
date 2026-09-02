DROP INDEX IF EXISTS idx_candidates_overall_score;

ALTER TABLE candidates
    DROP COLUMN IF EXISTS last_scored_at,
    DROP COLUMN IF EXISTS missing_skills,
    DROP COLUMN IF EXISTS matched_skills,
    DROP COLUMN IF EXISTS score_breakdown,
    DROP COLUMN IF EXISTS overall_score;
