-- Extend candidates table for full candidate management + future AI fields

ALTER TABLE candidates ADD COLUMN IF NOT EXISTS job_id UUID REFERENCES jobs (id) ON DELETE SET NULL;
ALTER TABLE candidates ADD COLUMN IF NOT EXISTS name VARCHAR(255);
ALTER TABLE candidates ADD COLUMN IF NOT EXISTS experience_years INTEGER;
ALTER TABLE candidates ADD COLUMN IF NOT EXISTS current_company VARCHAR(255);
ALTER TABLE candidates ADD COLUMN IF NOT EXISTS current_designation VARCHAR(255);
ALTER TABLE candidates ADD COLUMN IF NOT EXISTS location VARCHAR(255);
ALTER TABLE candidates ADD COLUMN IF NOT EXISTS skills TEXT[] NOT NULL DEFAULT '{}';
ALTER TABLE candidates ADD COLUMN IF NOT EXISTS resume_url VARCHAR(512);
ALTER TABLE candidates ADD COLUMN IF NOT EXISTS resume_text TEXT;
ALTER TABLE candidates ADD COLUMN IF NOT EXISTS resume_summary TEXT;
ALTER TABLE candidates ADD COLUMN IF NOT EXISTS parsing_status VARCHAR(50) NOT NULL DEFAULT 'pending';
ALTER TABLE candidates ADD COLUMN IF NOT EXISTS embedding_status VARCHAR(50) NOT NULL DEFAULT 'pending';

UPDATE candidates
SET name = trim(first_name || ' ' || last_name)
WHERE name IS NULL AND first_name IS NOT NULL;

UPDATE candidates SET status = 'applied' WHERE status = 'new';
UPDATE candidates SET status = 'rejected' WHERE status = 'withdrawn';

ALTER TABLE candidates DROP CONSTRAINT IF EXISTS candidates_status_check;
ALTER TABLE candidates ADD CONSTRAINT candidates_status_check CHECK (
    status IN ('applied', 'screening', 'shortlisted', 'recruiter_shortlisted', 'ai_shortlisted', 'interview', 'selected', 'offer', 'hired', 'rejected')
);

ALTER TABLE candidates ALTER COLUMN status SET DEFAULT 'applied';

ALTER TABLE candidates DROP CONSTRAINT IF EXISTS candidates_first_name_not_empty;
ALTER TABLE candidates DROP CONSTRAINT IF EXISTS candidates_last_name_not_empty;

ALTER TABLE candidates DROP COLUMN IF EXISTS first_name;
ALTER TABLE candidates DROP COLUMN IF EXISTS last_name;
ALTER TABLE candidates DROP COLUMN IF EXISTS linkedin_url;
ALTER TABLE candidates DROP COLUMN IF EXISTS notes;

ALTER TABLE candidates ALTER COLUMN name SET NOT NULL;

ALTER TABLE candidates DROP CONSTRAINT IF EXISTS candidates_parsing_status_check;
ALTER TABLE candidates ADD CONSTRAINT candidates_parsing_status_check CHECK (
    parsing_status IN ('pending', 'processing', 'completed', 'failed')
);

ALTER TABLE candidates DROP CONSTRAINT IF EXISTS candidates_embedding_status_check;
ALTER TABLE candidates ADD CONSTRAINT candidates_embedding_status_check CHECK (
    embedding_status IN ('pending', 'processing', 'completed', 'failed')
);

CREATE INDEX IF NOT EXISTS idx_candidates_job_id ON candidates (job_id);
CREATE INDEX IF NOT EXISTS idx_candidates_name ON candidates (company_id, lower(name));
CREATE INDEX IF NOT EXISTS idx_candidates_email ON candidates (company_id, lower(email));

DROP INDEX IF EXISTS idx_candidates_active_pipeline;
CREATE INDEX idx_candidates_active_pipeline ON candidates (company_id, updated_at DESC)
    WHERE status IN ('applied', 'screening', 'shortlisted', 'interview', 'offer');
