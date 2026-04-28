ALTER TABLE servers
    DROP COLUMN IF EXISTS ssh_password,
    DROP COLUMN IF EXISTS ssh_user;
