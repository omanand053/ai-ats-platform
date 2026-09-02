DROP INDEX IF EXISTS idx_candidates_active_pipeline;
CREATE INDEX idx_candidates_active_pipeline ON candidates (company_id, updated_at DESC)
    WHERE status IN ('new', 'screening', 'interview', 'offer');

ALTER TABLE candidates DROP CONSTRAINT IF EXISTS candidates_embedding_status_check;
ALTER TABLE candidates DROP CONSTRAINT IF EXISTS candidates_parsing_status_check;

ALTER TABLE candidates ADD COLUMN IF NOT EXISTS first_name VARCHAR(100);
ALTER TABLE candidates ADD COLUMN IF NOT EXISTS last_name VARCHAR(100);
ALTER TABLE candidates ADD COLUMN IF NOT EXISTS linkedin_url VARCHAR(512);
ALTER TABLE candidates ADD COLUMN IF NOT EXISTS notes TEXT;

UPDATE candidates SET
    first_name = split_part(name, ' ', 1),
    last_name = nullif(trim(substring(name from position(' ' in name) + 1)), '')
WHERE name IS NOT NULL AND first_name IS NULL;

ALTER TABLE candidates DROP COLUMN IF EXISTS parsing_status;
ALTER TABLE candidates DROP COLUMN IF EXISTS embedding_status;
ALTER TABLE candidates DROP COLUMN IF EXISTS resume_summary;
ALTER TABLE candidates DROP COLUMN IF EXISTS resume_text;
ALTER TABLE candidates DROP COLUMN IF EXISTS resume_url;
ALTER TABLE candidates DROP COLUMN IF EXISTS skills;
ALTER TABLE candidates DROP COLUMN IF EXISTS location;
ALTER TABLE candidates DROP COLUMN IF EXISTS current_designation;
ALTER TABLE candidates DROP COLUMN IF EXISTS current_company;
ALTER TABLE candidates DROP COLUMN IF EXISTS experience_years;
ALTER TABLE candidates DROP COLUMN IF EXISTS job_id;
ALTER TABLE candidates DROP COLUMN IF EXISTS name;

DROP INDEX IF EXISTS idx_candidates_email;
DROP INDEX IF EXISTS idx_candidates_name;
DROP INDEX IF EXISTS idx_candidates_job_id;

ALTER TABLE candidates DROP CONSTRAINT IF EXISTS candidates_status_check;
ALTER TABLE candidates ADD CONSTRAINT candidates_status_check CHECK (
    status IN ('new', 'screening', 'interview', 'offer', 'hired', 'rejected', 'withdrawn')
);

ALTER TABLE candidates ALTER COLUMN status SET DEFAULT 'new';

ALTER TABLE candidates ADD CONSTRAINT candidates_first_name_not_empty CHECK (char_length(trim(first_name)) > 0);
ALTER TABLE candidates ADD CONSTRAINT candidates_last_name_not_empty CHECK (char_length(trim(last_name)) > 0);
