# Release Notes — v0.1.15

## Logout & Session Integrity

### What's new

- **Logout button** (already present in sidebar) now correctly clears the token and redirects to `/login`.
- **Token expiry validation**: the dashboard layout decodes the JWT and checks `exp` on every page load. An expired session redirects to login instantly instead of waiting for the first API call to fail.
- **401/403 cleanup**: `apiFetch` now calls `clearToken()` before redirecting on auth errors, so stale tokens are always removed.

### Bug fix

JWT claim mismatch between the auth service and the middleware caused a redirect loop immediately after login:

- Auth service was issuing `sub` / `team_id` claims.
- Middleware `Claims` struct mapped only `uid` / `tid` → `TeamID` was always empty → every API call returned 403 → `apiFetch` redirected to `/login`.

Fixed by:
1. `issueToken` now emits `uid` + `tid`.
2. `Claims` struct adds `TeamIDLegacy json:"team_id"` as a backward-compat fallback.

### Upgrade

Frontend-only change (+ Go auth middleware). Rebuild both services:

```bash
git pull --ff-only
docker compose \
  -f deployments/docker/docker-compose.yaml \
  -f deployments/docker/docker-compose.override.yaml \
  up -d --build api frontend
```
