CREATE TABLE IF NOT EXISTS task_logs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    site_id     UUID        NOT NULL,
    task_type   VARCHAR(64) NOT NULL DEFAULT 'deploy',
    status      VARCHAR(32) NOT NULL DEFAULT 'running',
    output      TEXT        NOT NULL DEFAULT '',
    started_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_task_logs_site_id ON task_logs (site_id, started_at DESC);
