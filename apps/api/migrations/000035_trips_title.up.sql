-- User-editable trip title (AI-generated or hand-typed).
-- Nullable so existing trips are unaffected; display falls back to reach name.
ALTER TABLE trips ADD COLUMN title TEXT;
