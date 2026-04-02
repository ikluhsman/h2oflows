-- reach_embeddings stores vector embeddings for RAG queries.
-- Each row is one "chunk" of reach content — a rapid description,
-- access point, reach overview, or flow range summary.
-- The embedding dimension is 1536 (text-embedding-3-small).

CREATE TABLE reach_embeddings (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    reach_id    uuid NOT NULL REFERENCES reaches(id) ON DELETE CASCADE,

    -- Source record — exactly one of these is set.
    rapid_id    uuid REFERENCES rapids(id)       ON DELETE CASCADE,
    access_id   uuid REFERENCES reach_access(id) ON DELETE CASCADE,

    -- What type of content this chunk represents.
    chunk_type  text NOT NULL CHECK (chunk_type IN (
        'reach_description',
        'rapid',
        'access_point',
        'flow_ranges'
    )),

    -- The plain text that was embedded — stored so we can include it
    -- as context in the Claude prompt without re-fetching.
    content     text NOT NULL,

    -- The embedding vector. 1536 dimensions = text-embedding-3-small.
    embedding   vector(1536) NOT NULL,

    created_at  timestamptz NOT NULL DEFAULT NOW(),
    updated_at  timestamptz NOT NULL DEFAULT NOW()
);

-- IVFFlat index for fast approximate nearest-neighbour search.
-- lists=100 is appropriate for tables up to ~1M rows.
CREATE INDEX reach_embeddings_embedding_idx
    ON reach_embeddings
    USING ivfflat (embedding vector_cosine_ops)
    WITH (lists = 100);

-- Filter index — used when scoping queries to a specific reach.
CREATE INDEX reach_embeddings_reach_id_idx ON reach_embeddings (reach_id);
