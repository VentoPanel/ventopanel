## v0.1.7

### Highlights

- Added frontend under `frontend/` — Next.js 15, Tailwind CSS, Shadcn UI, TanStack Query v5.
- Dashboard with server/site stat cards and data tables.
- Login page with JWT token input (stored in `localStorage`).
- Route guard: unauthenticated users are redirected to `/login`.
- Status badges colored by server/site state (`connected` → green, `error` → red, etc.).
- `frontend/Dockerfile` with multi-stage build; standalone Next.js output for minimal image size.
- `frontend` service in Docker Compose (port 3000) with log rotation and restart policy.

### Running

```bash
# local dev
cd frontend
npm install
NEXT_PUBLIC_API_URL=http://localhost:8080 npm run dev

# via Docker Compose
docker compose \
  -f deployments/docker/docker-compose.yaml \
  -f deployments/docker/docker-compose.override.yaml \
  up -d --build
```

Open http://localhost:3000, paste your JWT token on the login page.
