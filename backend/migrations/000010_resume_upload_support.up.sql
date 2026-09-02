-- Allow resume upload before candidate creation; track company and storage path.

ALTER TABLE resumes ADD COLUMN IF NOT EXISTS company_id UUID REFERENCES companies (id) ON DELETE CASCADE;
ALTER TABLE resumes ADD COLUMN IF NOT EXISTS storage_path VARCHAR(1024);
ALTER TABLE resumes ALTER COLUMN candidate_id DROP NOT NULL;

CREATE INDEX IF NOT EXISTS idx_resumes_company_id ON resumes (company_id);
