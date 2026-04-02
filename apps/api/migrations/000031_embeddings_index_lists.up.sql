-- Drop the IVFFlat index entirely for now.
-- At <1000 rows, pgvector performs an exact sequential scan which is faster
-- and more accurate than approximate IVFFlat search.
-- Re-add an HNSW index when reach_embeddings grows past ~5000 rows.

DROP INDEX IF EXISTS reach_embeddings_embedding_idx;
