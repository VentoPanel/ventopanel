CREATE TABLE IF NOT EXISTS app_settings (
    key        TEXT PRIMARY KEY,
    value      TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Seed empty defaults so GET always returns all expected keys.
INSERT INTO app_settings (key, value) VALUES
    ('telegram_bot_token', ''),
    ('telegram_chat_id',   ''),
    ('whatsapp_webhook_url', '')
ON CONFLICT (key) DO NOTHING;
