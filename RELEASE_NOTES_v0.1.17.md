# Release Notes — v0.1.17

## Deployment Logs

Every site deploy run is now fully recorded with its output.

### What's new

**Backend**
- New `task_logs` table (migration applied automatically on startup).
- Each deploy captures stdout/stderr of every SSH step (`mkdir`, `docker compose up`, `nginx -t`, etc.) using `RunOutput`.
- Log entry created at deploy start, updated with final status and output on completion.
- `GET /api/v1/sites/:id/logs?limit=20` returns the history.

**Frontend (Site Detail page)**
- New **Deploy Logs** section above the Event History timeline.
- Each run shows: status badge (green/red/blue), short ID, relative time, duration in seconds.
- Click any row to expand and read the full SSH output in a monospace block.
- Auto-refreshes every 15 s.

### Upgrade

Both API and frontend changed. Rebuild both:

```bash
git pull --ff-only
docker compose \
  -f deployments/docker/docker-compose.yaml \
  -f deployments/docker/docker-compose.override.yaml \
  up -d --build api frontend
```

The migration adds the `task_logs` table automatically on API startup — no manual SQL needed.
