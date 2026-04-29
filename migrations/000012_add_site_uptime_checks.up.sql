CREATE TABLE IF NOT EXISTS site_uptime_checks (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    site_id     UUID        NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    checked_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    status      TEXT        NOT NULL, -- 'up' | 'down'
    latency_ms  INT,
    status_code INT,
    error       TEXT
);

CREATE INDEX IF NOT EXISTS idx_uptime_site_checked
    ON site_uptime_checks(site_id, checked_at DESC);
