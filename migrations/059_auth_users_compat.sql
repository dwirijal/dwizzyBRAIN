ALTER TABLE users
  ADD COLUMN IF NOT EXISTS username TEXT,
  ADD COLUMN IF NOT EXISTS display_name TEXT,
  ADD COLUMN IF NOT EXISTS avatar_url TEXT,
  ADD COLUMN IF NOT EXISTS timezone TEXT,
  ADD COLUMN IF NOT EXISTS locale TEXT,
  ADD COLUMN IF NOT EXISTS plan_override TEXT;

UPDATE users
SET username = COALESCE(
      NULLIF(username, ''),
      NULLIF(name, ''),
      NULLIF(split_part(COALESCE(email, ''), '@', 1), ''),
      'user_' || substr(id::text, 1, 8)
    ),
    display_name = COALESCE(NULLIF(display_name, ''), NULLIF(name, ''), NULLIF(username, ''), 'dwizzy user'),
    avatar_url = COALESCE(NULLIF(avatar_url, ''), NULLIF(picture, '')),
    timezone = COALESCE(NULLIF(timezone, ''), 'UTC'),
    locale = COALESCE(NULLIF(locale, ''), 'id-ID'),
    plan_override = COALESCE(
      NULLIF(plan_override, ''),
      CASE WHEN COALESCE(is_premium, false) THEN 'premium' ELSE 'free' END
    )
WHERE TRUE;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_indexes
    WHERE schemaname = current_schema()
      AND indexname = 'users_username_key'
  ) THEN
    EXECUTE 'CREATE UNIQUE INDEX users_username_key ON users (username)';
  END IF;
END $$;

UPDATE schema_migrations
SET dirty = false
WHERE version = 59;

