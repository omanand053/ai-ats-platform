CREATE TABLE resumes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    candidate_id UUID NOT NULL REFERENCES candidates (id) ON DELETE CASCADE,
    uploaded_by UUID REFERENCES users (id) ON DELETE SET NULL,
    file_name VARCHAR(255) NOT NULL,
    file_url VARCHAR(1024) NOT NULL,
    file_size BIGINT,
    mime_type VARCHAR(127),
    parsed_text TEXT,
    is_primary BOOLEAN NOT NULL DEFAULT FALSE,
    parsing_status VARCHAR(50) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT resumes_file_name_not_empty CHECK (char_length(trim(file_name)) > 0),
    CONSTRAINT resumes_file_url_not_empty CHECK (char_length(trim(file_url)) > 0),
    CONSTRAINT resumes_file_size_check CHECK (file_size IS NULL OR file_size >= 0),
    CONSTRAINT resumes_parsing_status_check CHECK (
        parsing_status IN ('pending', 'processing', 'completed', 'failed')
    )
);

CREATE INDEX idx_resumes_candidate_id ON resumes (candidate_id);
CREATE INDEX idx_resumes_uploaded_by ON resumes (uploaded_by);
CREATE INDEX idx_resumes_candidate_primary ON resumes (candidate_id) WHERE is_primary = TRUE;
CREATE INDEX idx_resumes_parsing_pending ON resumes (parsing_status, created_at)
    WHERE parsing_status IN ('pending', 'processing');

CREATE TRIGGER set_resumes_updated_at
    BEFORE UPDATE ON resumes
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();
