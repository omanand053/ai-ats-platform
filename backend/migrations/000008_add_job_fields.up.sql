ALTER TABLE jobs
    ADD COLUMN experience_required VARCHAR(255),
    ADD COLUMN required_skills TEXT[] NOT NULL DEFAULT '{}';

CREATE INDEX idx_jobs_required_skills ON jobs USING GIN (required_skills);
