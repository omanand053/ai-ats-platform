DROP INDEX IF EXISTS idx_resumes_company_id;
ALTER TABLE resumes DROP COLUMN IF EXISTS storage_path;
ALTER TABLE resumes DROP COLUMN IF EXISTS company_id;
