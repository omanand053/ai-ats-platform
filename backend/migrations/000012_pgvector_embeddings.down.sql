DROP TRIGGER IF EXISTS set_embeddings_updated_at ON embeddings;
DROP INDEX IF EXISTS idx_embeddings_hnsw;
DROP INDEX IF EXISTS idx_embeddings_entity;
DROP INDEX IF EXISTS idx_embeddings_company_entity;
DROP TABLE IF EXISTS embeddings;

ALTER TABLE jobs DROP CONSTRAINT IF EXISTS jobs_embedding_status_check;
ALTER TABLE jobs DROP COLUMN IF EXISTS embedding_status;

-- Extension left in place (may be used by other objects).
