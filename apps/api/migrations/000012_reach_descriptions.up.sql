-- Long-form reach description (markdown). Separate from the schema columns on reaches
-- so provenance and verification are tracked independently.
ALTER TABLE reaches
    ADD COLUMN description              TEXT,
    ADD COLUMN description_source       TEXT DEFAULT 'ai_seed'
                                        CHECK (description_source IN ('ai_seed','community','maintainer')),
    ADD COLUMN description_ai_confidence SMALLINT CHECK (description_ai_confidence BETWEEN 0 AND 100),
    ADD COLUMN description_verified     BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN description_updated_at   TIMESTAMPTZ;
