CREATE TABLE IF NOT EXISTS site_env_vars (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    site_id    UUID        NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    key        TEXT        NOT NULL,
    value_enc  TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (site_id, key)
);

CREATE INDEX IF NOT EXISTS idx_site_env_vars_site_id ON site_env_vars(site_id);
