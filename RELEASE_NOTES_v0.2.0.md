# Release Notes — v0.2.0 "MVP"

This is the first complete milestone release of **VentoPanel** — a self-hosted control panel
for managing servers, deploying sites, and monitoring infrastructure via SSH.

---

## What's included

### Authentication & Access Control
- Email + password login with bcrypt hashing
- JWT-based sessions with expiry validation (12 h TTL)
- First registered user automatically becomes admin (bootstrap)
- Role system: **admin** · **editor** · **viewer**
- Role-based UI: write/delete actions hidden by role
- Logout clears token; expired tokens redirect to login automatically

### Server Management
- Add / edit / delete servers with SSH credentials (AES-GCM encrypted at rest)
- Connect and provision servers over SSH
- Live monitoring: CPU cores, load average, RAM usage, disk usage, uptime
- Server detail page with auto-refreshing stats (30 s)

### Site Management
- Add / edit / delete sites linked to servers
- Deploy via Docker Compose + Nginx (Node.js and PHP runtimes)
- SSL certificates via Let's Encrypt / Certbot
- SSL status card: expiry date, days remaining, color-coded (valid / expiring / expired)
- Per-site SSL Renew button
- Site detail page with audit timeline and deploy log history
- Deploy logs capture stdout/stderr of each SSH step with expandable view

### Dashboard
- Summary stat cards: servers, sites, error counts
- Recent Activity feed (last 8 audit events)
- Quick lists linking to server and site detail pages

### Audit Log
- Status change events with cursor-based pagination
- Filter by resource type (server / site)
- Color-coded status badges

### Notifications
- Telegram Bot API integration (HTML messages)
- WhatsApp generic webhook support (Evolution API / Twilio compatible)
- Configurable via **Settings** page — no restart needed after change

### User Management
- List all team members with roles and join dates
- Change role inline (admin only)
- Delete users with self-deletion guard (admin only)

### Observability
- Prometheus metrics: SSL renew operations, ACL denied decisions
- Grafana dashboard provisioning
- ACL deny audit events written to `status_events`

### Infrastructure
- Docker Compose stack: API · PostgreSQL · Redis · Frontend
- Auto-migrations on API startup (`golang-migrate`)
- Background task queue via Asynq
- Distributed locks for idempotent deploys/provisions
- GitHub Actions CI: lint, tests, OpenAPI validation

---

## Release summary (v0.1.0 → v0.2.0)

| Version | Feature |
|---|---|
| v0.1.0 | Core backend: servers, sites, SSL, audit, Prometheus |
| v0.1.4 | ACL deny metrics and audit events |
| v0.1.7 | Next.js frontend scaffold |
| v0.1.8 | Full CRUD UI with modals |
| v0.1.9 | Auto-refresh (15 s polling) |
| v0.1.10 | Telegram + WhatsApp alerts |
| v0.1.11 | Audit Log UI |
| v0.1.12 | Server monitoring (SSH stats) |
| v0.1.13 | Site detail page |
| v0.1.14 | Email + password authentication |
| v0.1.15 | Logout and session integrity |
| v0.1.16 | Dashboard overview |
| v0.1.17 | Deployment logs with SSH output |
| v0.1.18 | Settings page (notifications via UI) |
| v0.1.19 | User management UI |
| v0.1.20 | Role-based UI |
| v0.1.21 | SSL status card and per-site Renew |
| **v0.2.0** | **MVP milestone release** |

---

## Deploying

```bash
git clone https://github.com/VentoPanel/ventopanel.git
cd ventopanel
cp .env.example .env
# edit .env with your values
docker compose \
  -f deployments/docker/docker-compose.yaml \
  up -d --build
```

Open `http://your-server:3000` → register the first user (becomes admin automatically).

---

## What's next (v0.3+)

- Invite flow with email verification
- Multi-team support
- Real-time log streaming (SSE/WebSocket)
- Git-triggered auto-deploy (webhooks)
- Backup management
