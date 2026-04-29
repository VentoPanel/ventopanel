-- API tokens for programmatic access (CI/CD, scripts)
CREATE TABLE IF NOT EXISTS api_tokens (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name         TEXT        NOT NULL,
    token_hash   TEXT        NOT NULL UNIQUE,
    last_used_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_api_tokens_user_id  ON api_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_api_tokens_hash     ON api_tokens(token_hash);

-- TOTP (2FA) columns on users
ALTER TABLE users ADD COLUMN IF NOT EXISTS totp_secret  TEXT    NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS totp_enabled BOOLEAN NOT NULL DEFAULT false;
