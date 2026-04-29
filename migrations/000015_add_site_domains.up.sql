CREATE TABLE IF NOT EXISTS site_domains (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    site_id    UUID        NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    domain     TEXT        NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_site_domains_site_id ON site_domains(site_id);
