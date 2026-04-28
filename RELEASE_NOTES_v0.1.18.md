# Release Notes — v0.1.18

## Settings Page — Notification Credentials via UI

Telegram and WhatsApp credentials can now be managed through the web interface without editing environment variables or restarting containers.

### What's new

**Backend**
- `app_settings` table — key-value store. Seeded with three empty rows on first run.
- On startup: if env vars (`TELEGRAM_BOT_TOKEN`, etc.) are set and DB rows are still empty, they are automatically copied to DB.
- `PATCH /api/v1/settings/notifications` — saves new credentials instantly.
- `GET /api/v1/settings/notifications` — returns current config; bot token is masked.
- `AlertService` now reads config from DB on every `NotifyAll` call — changes take effect on the next alert without restarting.

**Frontend**
- New **Settings** page (sidebar link).
- Telegram section: bot token (show/hide toggle) + chat ID with instructions.
- WhatsApp section: webhook URL field.
- Save button with success/error toast.

### Upgrade

Both API and frontend changed. Rebuild both:

```bash
git pull --ff-only
docker compose \
  -f deployments/docker/docker-compose.yaml \
  -f deployments/docker/docker-compose.override.yaml \
  up -d --build api frontend
```

The `app_settings` migration runs automatically. Existing env-based credentials are preserved (seeded to DB on first startup).
