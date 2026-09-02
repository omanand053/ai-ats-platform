-- Phase 5 enterprise tables

CREATE TABLE IF NOT EXISTS company_ai_settings (
    company_id UUID PRIMARY KEY REFERENCES companies (id) ON DELETE CASCADE,
    weight_semantic DOUBLE PRECISION NOT NULL DEFAULT 0.40,
    weight_skills DOUBLE PRECISION NOT NULL DEFAULT 0.25,
    weight_experience DOUBLE PRECISION NOT NULL DEFAULT 0.15,
    weight_education DOUBLE PRECISION NOT NULL DEFAULT 0.10,
    weight_projects DOUBLE PRECISION NOT NULL DEFAULT 0.10,
    confidence_threshold DOUBLE PRECISION NOT NULL DEFAULT 55,
    eligibility_threshold DOUBLE PRECISION NOT NULL DEFAULT 40,
    updated_by UUID REFERENCES users (id) ON DELETE SET NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies (id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    type VARCHAR(80) NOT NULL,
    title TEXT NOT NULL,
    body TEXT NOT NULL DEFAULT '',
    entity_type VARCHAR(50),
    entity_id UUID,
    read_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_notifications_user
    ON notifications (company_id, user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies (id) ON DELETE CASCADE,
    actor_user_id UUID REFERENCES users (id) ON DELETE SET NULL,
    action VARCHAR(120) NOT NULL,
    resource_type VARCHAR(80) NOT NULL,
    resource_id UUID,
    meta JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_audit_logs_company
    ON audit_logs (company_id, created_at DESC);

CREATE TABLE IF NOT EXISTS interviews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies (id) ON DELETE CASCADE,
    candidate_id UUID NOT NULL REFERENCES candidates (id) ON DELETE CASCADE,
    job_id UUID REFERENCES jobs (id) ON DELETE SET NULL,
    title TEXT NOT NULL DEFAULT 'Interview',
    scheduled_at TIMESTAMPTZ NOT NULL,
    duration_minutes INTEGER NOT NULL DEFAULT 45,
    timezone VARCHAR(80) NOT NULL DEFAULT 'UTC',
    location TEXT,
    meeting_url TEXT,
    status VARCHAR(40) NOT NULL DEFAULT 'scheduled',
    interviewer_user_id UUID REFERENCES users (id) ON DELETE SET NULL,
    notes TEXT,
    created_by UUID REFERENCES users (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT interviews_status_check CHECK (status IN ('scheduled', 'completed', 'cancelled', 'no_show'))
);
CREATE INDEX IF NOT EXISTS idx_interviews_company_time
    ON interviews (company_id, scheduled_at ASC);

CREATE TABLE IF NOT EXISTS collaboration_comments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies (id) ON DELETE CASCADE,
    candidate_id UUID NOT NULL REFERENCES candidates (id) ON DELETE CASCADE,
    author_user_id UUID REFERENCES users (id) ON DELETE SET NULL,
    body TEXT NOT NULL,
    mentions UUID[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_collab_comments_candidate
    ON collaboration_comments (company_id, candidate_id, created_at DESC);

ALTER TABLE candidates ADD COLUMN IF NOT EXISTS assigned_to UUID REFERENCES users (id) ON DELETE SET NULL;

-- Expand roles for interviewer + keep viewer as read-only
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check
    CHECK (role IN ('admin', 'recruiter', 'hiring_manager', 'interviewer', 'viewer'));
