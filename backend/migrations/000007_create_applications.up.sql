CREATE TABLE applications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    candidate_id UUID NOT NULL REFERENCES candidates (id) ON DELETE CASCADE,
    job_id UUID NOT NULL REFERENCES jobs (id) ON DELETE CASCADE,
    status VARCHAR(50) NOT NULL DEFAULT 'applied',
    source VARCHAR(100),
    notes TEXT,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT applications_status_check CHECK (
        status IN ('applied', 'screening', 'interview', 'offer', 'rejected', 'hired')
    )
);

CREATE INDEX idx_applications_candidate_id ON applications (candidate_id);
CREATE INDEX idx_applications_job_id ON applications (job_id);
CREATE INDEX idx_applications_status ON applications (status);
CREATE INDEX idx_applications_not_deleted ON applications (deleted_at) WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX idx_applications_candidate_job_unique
    ON applications (candidate_id, job_id)
    WHERE deleted_at IS NULL;

CREATE TRIGGER set_applications_updated_at
    BEFORE UPDATE ON applications
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();
