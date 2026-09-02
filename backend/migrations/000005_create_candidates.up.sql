CREATE TABLE candidates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies (id) ON DELETE CASCADE,
    email VARCHAR(320) NOT NULL,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    phone VARCHAR(50),
    linkedin_url VARCHAR(512),
    source VARCHAR(100),
    status VARCHAR(50) NOT NULL DEFAULT 'new',
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT candidates_company_email_unique UNIQUE (company_id, email),
    CONSTRAINT candidates_status_check CHECK (
        status IN ('new', 'screening', 'interview', 'offer', 'hired', 'rejected', 'withdrawn')
    ),
    CONSTRAINT candidates_first_name_not_empty CHECK (char_length(trim(first_name)) > 0),
    CONSTRAINT candidates_last_name_not_empty CHECK (char_length(trim(last_name)) > 0)
);

CREATE INDEX idx_candidates_company_id ON candidates (company_id);
CREATE INDEX idx_candidates_company_status ON candidates (company_id, status);
CREATE INDEX idx_candidates_active_pipeline ON candidates (company_id, updated_at DESC)
    WHERE status IN ('new', 'screening', 'interview', 'offer');

CREATE TRIGGER set_candidates_updated_at
    BEFORE UPDATE ON candidates
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();
