# Release Notes — v0.1.19

## User Management UI

Team members can now be managed through the web interface.

### What's new

**Backend**
- `List` — returns all users ordered by creation date.
- `UpdateRole` — changes user role (`admin` / `editor` / `viewer`); validates allowed values.
- `Delete` — removes user; returns 400 if you attempt to delete your own account.

**Frontend — `/users` page**
- Table of all team members with email, relative join date, and role badge.
- Role badges: purple (admin), blue (editor), gray (viewer).
- Inline `<select>` to change role — updates instantly on change.
- Delete button opens a confirmation dialog.
- "Users" link added to the sidebar navigation.

### Upgrade

Both API and frontend changed. Rebuild both:

```bash
git pull --ff-only
docker compose \
  -f deployments/docker/docker-compose.yaml \
  -f deployments/docker/docker-compose.override.yaml \
  up -d --build api frontend
```

No new migrations — uses the existing `users` table from v0.1.14.
