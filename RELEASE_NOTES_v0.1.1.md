# VentoPanel v0.1.1

Security and usability patch release focused on access control hardening and developer auth ergonomics.

## Highlights

- Added JWT auth context middleware for request identity (`uid` and `tid` claims).
- Kept backward-compatible header fallback mode for gradual rollout (`AUTH_ALLOW_HEADER_FALLBACK`).
- Added development-only token minting endpoint (`POST /api/v1/dev/token`) for local testing without external IdP.
- Hardened ACL behavior on list endpoints to prevent data leakage.

## Access Control Improvements

- `GET /api/v1/sites` now returns only sites accessible by the authenticated team.
- `GET /api/v1/servers` now returns only servers accessible by the authenticated team via granted sites.
- Existing by-id ACL checks remain enforced for read/write flows on site and server resources.

## Configuration

- `AUTH_JWT_SECRET`: JWT signing/verification secret for auth middleware.
- `AUTH_ALLOW_HEADER_FALLBACK`:
  - `true` for compatibility during migration
  - recommended `false` in production to require JWT-derived identity

## Testing and Quality

- Added/expanded tests for:
  - JWT middleware behavior
  - dev token endpoint behavior (enabled/disabled)
  - ACL list filtering for sites and servers
- Full test suite remains green.
