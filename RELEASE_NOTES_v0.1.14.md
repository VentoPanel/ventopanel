## v0.1.14 — Email + Password Authentication

### Highlights

- **Real login**: `POST /api/v1/auth/login` accepts `{ email, password }` and returns a signed JWT — no more pasting tokens manually.
- **Registration**: `POST /api/v1/auth/register` creates a user; the **first user** automatically gets the `admin` role (bootstrap).
- **bcrypt** (cost 12) for secure password hashing.
- JWT TTL: **12 hours**, same payload format as before (`team_id`, `role`, `iss`, `aud`).
- New **`/login` page**: clean card with Sign In / Register tabs, show/hide password, loading spinner, inline errors.

### Migration

Run migrations on your server after pulling:

```bash
docker compose \
  -f deployments/docker/docker-compose.yaml \
  -f deployments/docker/docker-compose.override.yaml \
  up -d --build
```

The API container automatically applies migration `000005_add_users`.

### First-time setup

1. Open the login page → Register tab.
2. Enter your email, password (min 8 chars), and a Team ID from the database.
3. Click **Create Account** — this first account becomes admin.
4. Switch to Sign In and log in normally.
