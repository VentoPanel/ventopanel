ALTER TABLE servers
    DROP COLUMN IF EXISTS last_renew_status,
    DROP COLUMN IF EXISTS last_renew_at;
