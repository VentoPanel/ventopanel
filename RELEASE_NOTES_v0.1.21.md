# Release Notes — v0.1.21

## SSL Status on Site Detail Page

Certificate expiry is now visible directly on the site detail page with a one-click Renew button.

### What's new

**Backend**
- `ssl.Service.GetCertInfo` — connects via SSH and runs:
  ```
  openssl x509 -enddate -noout -in /etc/letsencrypt/live/{domain}/fullchain.pem
  ```
  Parses the expiry date and returns `{ domain, expires_at, days_left, status }`.
- Status values: `valid` · `expiring_soon` (≤ 30 days) · `expired` · `no_cert`
- `GET /api/v1/sites/:id/ssl` — returns cert info (authenticated).
- `POST /api/v1/sites/:id/ssl/renew` — queues SSL issue task for the site.

**Frontend — Site Detail page**
- New **SSL Certificate** card with color-coded left border:
  - 🟢 Green — valid
  - 🟡 Yellow — expiring within 30 days
  - 🔴 Red — expired or no cert found
- Shows: status · expiry date · days remaining (red ≤ 14, yellow ≤ 30, green > 30)
- **Renew** button visible for admin/editor roles.
- Card refreshes every 60 s in background.

### Upgrade

Both API and frontend changed. Rebuild both:

```bash
git pull --ff-only
docker compose \
  -f deployments/docker/docker-compose.yaml \
  -f deployments/docker/docker-compose.override.yaml \
  up -d --build api frontend
```

No new migrations needed.
