CREATE TABLE jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies (id) ON DELETE CASCADE,
    created_by UUID REFERENCES users (id) ON DELETE SET NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    department VARCHAR(100),
    location VARCHAR(255),
    employment_type VARCHAR(50) NOT NULL DEFAULT 'full_time',
    status VARCHAR(50) NOT NULL DEFAULT 'draft',
    requirements TEXT,
    salary_min INTEGER,
    salary_max INTEGER,
    currency CHAR(3) NOT NULL DEFAULT 'USD',
    published_at TIMESTAMPTZ,
    closed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT jobs_title_not_empty CHECK (char_length(trim(title)) > 0),
    CONSTRAINT jobs_employment_type_check CHECK (
        employment_type IN ('full_time', 'part_time', 'contract', 'internship', 'temporary')
    ),
    CONSTRAINT jobs_status_check CHECK (
        status IN ('draft', 'open', 'closed', 'archived')
    ),
    CONSTRAINT jobs_salary_range_check CHECK (
        salary_min IS NULL
        OR salary_max IS NULL
        OR salary_min <= salary_max
    )
);

CREATE INDEX idx_jobs_company_id ON jobs (company_id);
CREATE INDEX idx_jobs_created_by ON jobs (created_by);
CREATE INDEX idx_jobs_company_status ON jobs (company_id, status);
CREATE INDEX idx_jobs_open ON jobs (company_id, published_at DESC) WHERE status = 'open';

CREATE TRIGGER set_jobs_updated_at
    BEFORE UPDATE ON jobs
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();
