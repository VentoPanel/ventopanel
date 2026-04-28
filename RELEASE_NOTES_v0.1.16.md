# Release Notes — v0.1.16

## Dashboard Overview

The dashboard is now a real overview page instead of a duplicate of the servers/sites list pages.

### What's new

**4 stat cards (top row)**
| Card | Shows |
|---|---|
| Servers | total · connected · errors (red if >0) |
| Sites | total · deployed · errors (red if >0) |
| Recent Errors | count of error/failed/access_denied in last 8 audit events |
| Auto-refresh | current polling interval |

Cards for Servers, Sites, and Recent Errors are clickable links to their respective pages.

**Recent Activity feed**
- Last 8 audit events, refreshed every 30 s
- Resource type badge · `→` · status badge · reason text · "X ago" timestamp
- "View all" link to the full Audit Log page

**Quick lists (bottom row)**
- Servers preview (top 5) with status badges — each row links to the server detail page
- Sites preview (top 5) with status badges — each row links to the site detail page

### Upgrade

Frontend-only change. Rebuild frontend:

```bash
git pull --ff-only
docker compose \
  -f deployments/docker/docker-compose.yaml \
  -f deployments/docker/docker-compose.override.yaml \
  up -d --build frontend
```
