CREATE TABLE IF NOT EXISTS team_server_access (
    team_id   UUID        NOT NULL,
    server_id UUID        NOT NULL,
    role      VARCHAR(32) NOT NULL DEFAULT 'owner',
    PRIMARY KEY (team_id, server_id)
);

CREATE INDEX IF NOT EXISTS idx_team_server_access_server_id ON team_server_access (server_id);
