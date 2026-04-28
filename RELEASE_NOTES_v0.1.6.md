## v0.1.6

### Highlights

- Added `make smoke-prod-auth SITE_ID=<id>` for production-ready ACL/auth smoke verification.
- The target generates a JWT from `.env` (`AUTH_JWT_SECRET`, issuer, audience), executes an ACL deny request, checks `ventopanel_acl_denied_total`, and prints latest `access_denied` audit rows from Postgres.
- Updated `OPERATIONS.md` with the one-command shortcut.

### Notes

- Requires local `.env` with auth keys and a valid `SITE_ID` that has no grant for team `11111111-1111-1111-1111-111111111111`.
- Requires `python3`, `curl`, and Docker Compose available on the host.
