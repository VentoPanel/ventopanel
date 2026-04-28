# Release Notes — v0.1.20

## Role-based UI

The interface now respects the authenticated user's role. Destructive and administrative actions are hidden for lower-privilege users.

### Permission matrix

| Action | viewer | editor | admin |
|---|:---:|:---:|:---:|
| View all pages | ✓ | ✓ | ✓ |
| Create server/site | — | ✓ | ✓ |
| Edit server/site | — | ✓ | ✓ |
| Deploy site | — | ✓ | ✓ |
| Connect/Provision server | — | ✓ | ✓ |
| Delete server/site | — | — | ✓ |
| Change user roles | — | — | ✓ |
| Delete users | — | — | ✓ |

### Implementation

- `Claims.Role` extracted from JWT and stored in request context by the auth middleware.
- Frontend decodes role directly from the stored JWT (`getTokenPayload().role`) — no extra API call.
- `useAuth()` hook provides `isAdmin` and `canWrite` booleans to any component.

### Upgrade

Both API (middleware) and frontend changed. Rebuild both:

```bash
git pull --ff-only
docker compose \
  -f deployments/docker/docker-compose.yaml \
  -f deployments/docker/docker-compose.override.yaml \
  up -d --build api frontend
```

Users need to log out and back in once for the new token with the `role` claim to take effect.
