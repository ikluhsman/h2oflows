DROP INDEX IF EXISTS reach_embeddings_embedding_idx;

ALTER TABLE reach_embeddings
    ALTER COLUMN embedding TYPE vector(1536);

CREATE INDEX reach_embeddings_embedding_idx
    ON reach_embeddings
    USING ivfflat (embedding vector_cosine_ops)
    WITH (lists = 100);
