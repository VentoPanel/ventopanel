## v0.1.5

### Highlights

- Added `deployments/docker/docker-compose.override.yaml` for server-local production defaults.
- Enabled Docker log rotation (`json-file`, `max-size=10m`, `max-file=5`) for `api`, `postgres`, and `redis`.
- Updated `OPERATIONS.md` to use merged compose files and added ACL deny smoke checks (`health`, `metrics`, `status_events`).

### Notes

- Run stack commands with both compose files:
  - `docker compose -f deployments/docker/docker-compose.yaml -f deployments/docker/docker-compose.override.yaml ...`
- Keep override file server-local if desired; add it to local git exclude on hosts where it should remain untracked.
