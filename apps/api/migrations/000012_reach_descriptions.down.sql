ALTER TABLE reaches
    DROP COLUMN IF EXISTS description,
    DROP COLUMN IF EXISTS description_source,
    DROP COLUMN IF EXISTS description_ai_confidence,
    DROP COLUMN IF EXISTS description_verified,
    DROP COLUMN IF EXISTS description_updated_at;
