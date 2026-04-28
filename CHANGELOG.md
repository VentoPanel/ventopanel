# Changelog

All notable changes to this project are documented in this file.

## v0.1.14 - 2026-04-28

### Added

- `users` table migration (email, bcrypt password_hash, team_id, role).
- `domain/user` package with `User`, `Repository` interface, sentinel errors.
- `UserRepository` with `Create`, `GetByEmail`, `GetByID`, `Count`.
- `service/auth`: `Register` (first user → admin bootstrap) + `Login` with bcrypt + JWT issuing (12h TTL, same format as existing middleware).
- `POST /api/v1/auth/login` — { email, password } → { token, email, role }.
- `POST /api/v1/auth/register` — { email, password, team_id } → { id, email, role }.
- Frontend `/login` page replaced with email+password form (tab: Sign In / Register), show/hide password toggle, loading spinner, inline error and success messages.
- `login()` and `registerUser()` functions added to `lib/api.ts`.

## v0.1.13 - 2026-04-28

### Added

- `/sites/[id]` detail page: info cards (runtime, domain, repository, created date), status badge, Deploy / Edit / Delete actions, and a timeline of audit events scoped to this site.
- `fetchSiteByID` API function for fetching a single site by ID.
- Site names in the sites table are now clickable links to the detail page.
- Site detail auto-refreshes every 15 seconds; RefreshIndicator shown in header.
- Audit timeline shows `toStatus → fromStatus` with colour-coded badges and timestamps.

## v0.1.12 - 2026-04-28

### Added

- `GET /api/v1/servers/:id/stats` — new endpoint that SSHes into the server and returns live CPU cores, load average, RAM usage, disk usage, and uptime.
- `RunOutput` method on SSH executor returning command stdout/stderr as a string.
- `ServerStats` domain type.
- Frontend: `/servers/[id]` detail page with:
  - Server info cards (provider, SSH user, created date).
  - Live monitoring cards: CPU (cores + load), Memory (used/total bar), Disk (used/total bar, color-coded), Uptime.
  - Connect / Provision action buttons.
  - Auto-refresh every 30 seconds with `RefreshIndicator`.
- Server name in the servers table is now a clickable link to the detail page.

## v0.1.11 - 2026-04-28

### Added

- `/audit` page in the frontend: table of all status-change events with resource type, from/to status badges, reason, and timestamps.
- Color-coded status badges: red for failures/denied, green for success, blue for in-progress states.
- Filter by resource type (All / Server / Site).
- Cursor-based "Load more" pagination via TanStack Query `useInfiniteQuery`.
- "Audit Log" link added to the sidebar navigation.

## v0.1.10 - 2026-04-28

### Added

- Real Telegram Bot API integration: HTTP POST to `sendMessage` with HTML parse mode and 5-second timeout.
- WhatsApp generic webhook support (Evolution API / Twilio compatible).
- Success notifications alongside failure alerts for deploy, provision, SSL issue, and SSL renew tasks.
- `alertService.NotifyAll` now fans out to all notifiers and collects errors with `errors.Join` so one failing channel does not block others.
- Contextual HTML messages with site/server IDs and error details (e.g. `🚨 Site deploy FAILED / Site: <code>…</code>`).

### Changed

- `alert.NewService` signature changed to variadic `NewService(notifiers ...Notifier)`.
- Updated alert service test to reflect fan-out behaviour.

## v0.1.9 - 2026-04-28

### Added

- Auto-refresh for servers and sites tables every 15 seconds via TanStack Query `refetchInterval`.
- `RefreshIndicator` component showing live status: green dot when idle, spinner when fetching, "Updated Xs ago" counter, and manual refresh button.
- Dashboard, Servers, and Sites pages updated with refresh indicators.

## v0.1.8 - 2026-04-28

### Added

- Full CRUD UI for servers and sites via modal dialogs (create, edit, delete with confirmation).
- Action buttons in tables: Connect and Provision for servers; Deploy for sites.
- Toast notifications (Sonner) for all mutations: success and error feedback.
- New UI primitives: `dialog`, `input`, `label` (Shadcn/Radix-based).
- Mutation hooks: `use-server-mutations`, `use-site-mutations` (TanStack Query `useMutation`).
- API client extended with `createServer`, `updateServer`, `deleteServer`, `connectServer`, `provisionServer`, `createSite`, `updateSite`, `deleteSite`, `deploySite`.

## v0.1.7 - 2026-04-28

### Added

- Frontend scaffolding under `frontend/` using Next.js 15 App Router, TypeScript, Tailwind CSS 3, Shadcn UI primitives, and TanStack Query v5.
- Dashboard page with stat cards (server/site counts), servers table, and sites table.
- Servers and Sites list pages with status badges (`success`, `warning`, `destructive`).
- Login page: JWT token input stored in `localStorage`, redirects to dashboard.
- Route guard in dashboard layout: unauthenticated users are redirected to `/login`.
- API client in `frontend/lib/api.ts`: reads token from `localStorage`, adds `Authorization` header, redirects on 401/403.
- `frontend/Dockerfile` with multi-stage build (`node:20-alpine`) and standalone Next.js output.
- `frontend` service added to `deployments/docker/docker-compose.yaml` and `docker-compose.override.yaml` (port 3000, log rotation, restart policy).

## v0.1.6 - 2026-04-28

### Added

- Added `make smoke-prod-auth SITE_ID=<id>` to automate production auth smoke checks: health endpoint, ACL deny request, ACL deny metrics, and `status_events` audit verification.
- Documented the new Makefile shortcut in `OPERATIONS.md` for repeatable post-release verification on servers.

## v0.1.5 - 2026-04-28

### Added

- Added `deployments/docker/docker-compose.override.yaml` with production-safe defaults (`restart: unless-stopped`) and Docker json-file log rotation for `api`, `postgres`, and `redis`.
- Expanded `OPERATIONS.md` with merged compose commands (`base + override`) and a post-release ACL deny smoke procedure covering health, metrics, and audit verification.

## v0.1.4 - 2026-04-28

### Added

- ACL deny observability with Prometheus metric `ventopanel_acl_denied_total` labeled by resource and deny reason.
- ACL deny audit events for server/site authorization failures, written to `status_events` with `to_status=access_denied`.
- Integration test coverage to verify deny decisions are persisted in audit storage.

## v0.1.0 - 2026-04-28

### Added

- Production-oriented SSL workflow with async issue/renew tasks, scheduler, and observability metrics.
- Redis distributed locks and Asynq task deduplication for deploy/provision/SSL idempotency.
- Audit trail subsystem with status transition events, filterable read API, and cursor pagination.
- OpenAPI spec for `GET /api/v1/audit/status-events` in `openapi/audit-status-events.yaml`.
- Monitoring bundle with Prometheus, alert rules, and Grafana dashboard provisioning.
- Integration tests for audit API and server connect audit transitions.
- CI workflow with OpenAPI lint, Go lint, unit tests, and Postgres-backed integration tests.

### Changed

- Refactored backend toward clean architecture boundaries (`domain/service/repository/transport`).
- Hardened server credential handling with AES-GCM encryption at rest.
- Improved server and site lifecycle safety via explicit transition state machine validation.
- Expanded HTTP API surface for server/site CRUD, connect/provision/deploy/SSL, metrics, and observability.
- Updated docs and Makefile commands for monitoring, integration testing, and CI usage.

### Notes

- `total_count` in audit listing is optional (`include_total=true`) and intentionally ignores cursor pagination.
