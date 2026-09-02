CREATE EXTENSION IF NOT EXISTS vector;

ALTER TABLE jobs
    ADD COLUMN IF NOT EXISTS embedding_status VARCHAR(50) NOT NULL DEFAULT 'pending';

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'jobs_embedding_status_check'
    ) THEN
        ALTER TABLE jobs
            ADD CONSTRAINT jobs_embedding_status_check
            CHECK (embedding_status IN ('pending', 'processing', 'completed', 'failed'));
    END IF;
END $$;

CREATE TABLE IF NOT EXISTS embeddings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies (id) ON DELETE CASCADE,
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID NOT NULL,
    content_hash VARCHAR(64) NOT NULL,
    embedding vector(384) NOT NULL,
    embedding_model VARCHAR(100) NOT NULL,
    embedding_version VARCHAR(50) NOT NULL,
    embedded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    status VARCHAR(50) NOT NULL DEFAULT 'completed',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT embeddings_entity_type_check
        CHECK (entity_type IN ('resume', 'candidate', 'job')),
    CONSTRAINT embeddings_status_check
        CHECK (status IN ('pending', 'processing', 'completed', 'failed')),
    CONSTRAINT embeddings_entity_model_unique
        UNIQUE (entity_type, entity_id, embedding_model, embedding_version)
);

CREATE INDEX IF NOT EXISTS idx_embeddings_company_entity
    ON embeddings (company_id, entity_type, entity_id);

CREATE INDEX IF NOT EXISTS idx_embeddings_entity
    ON embeddings (entity_type, entity_id);

CREATE INDEX IF NOT EXISTS idx_embeddings_hnsw
    ON embeddings
    USING hnsw (embedding vector_cosine_ops);

DROP TRIGGER IF EXISTS set_embeddings_updated_at ON embeddings;
CREATE TRIGGER set_embeddings_updated_at
    BEFORE UPDATE ON embeddings
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();
