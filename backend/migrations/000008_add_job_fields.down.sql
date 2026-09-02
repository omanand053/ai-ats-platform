DROP INDEX IF EXISTS idx_jobs_required_skills;

ALTER TABLE jobs
    DROP COLUMN IF EXISTS required_skills,
    DROP COLUMN IF EXISTS experience_required;
