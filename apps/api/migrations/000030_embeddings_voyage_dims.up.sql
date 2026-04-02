-- Switch embedding column from 1536 (OpenAI text-embedding-3-small)
-- to 1024 (Voyage AI voyage-3), which is Anthropic's recommended RAG partner.
-- Table is empty at this point so the ALTER is instantaneous.

DROP INDEX IF EXISTS reach_embeddings_embedding_idx;

ALTER TABLE reach_embeddings
    ALTER COLUMN embedding TYPE vector(1024);

CREATE INDEX reach_embeddings_embedding_idx
    ON reach_embeddings
    USING ivfflat (embedding vector_cosine_ops)
    WITH (lists = 100);
