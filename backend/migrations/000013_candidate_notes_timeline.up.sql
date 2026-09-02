CREATE TABLE IF NOT EXISTS candidate_notes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies (id) ON DELETE CASCADE,
    candidate_id UUID NOT NULL REFERENCES candidates (id) ON DELETE CASCADE,
    author_user_id UUID REFERENCES users (id) ON DELETE SET NULL,
    body TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_candidate_notes_candidate
    ON candidate_notes (company_id, candidate_id, created_at DESC);

CREATE TABLE IF NOT EXISTS candidate_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies (id) ON DELETE CASCADE,
    candidate_id UUID NOT NULL REFERENCES candidates (id) ON DELETE CASCADE,
    event_type VARCHAR(80) NOT NULL,
    label TEXT NOT NULL,
    meta JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_candidate_events_candidate
    ON candidate_events (company_id, candidate_id, created_at ASC);
