# VentoPanel v0.1.2

Security hardening release focused on stronger JWT validation and safer production defaults.

## Highlights

- Added optional JWT `issuer` and `audience` verification in auth middleware.
- Introduced option-based auth middleware wiring for cleaner configuration and backward compatibility.
- Switched production-safe default for header fallback mode in config (`AUTH_ALLOW_HEADER_FALLBACK=false` by default).
- Added test coverage for valid/invalid issuer-audience claim handling.

## Configuration

- `AUTH_JWT_SECRET`: JWT signature verification secret.
- `AUTH_JWT_ISSUER`: expected `iss` claim (optional, recommended in production).
- `AUTH_JWT_AUDIENCE`: expected `aud` claim (optional, recommended in production).
- `AUTH_ALLOW_HEADER_FALLBACK`:
  - default in config is now `false` (safer baseline)
  - `.env.example` keeps `true` for local developer convenience and migration compatibility

## Notes

- Existing integrations are preserved through compatibility wrappers, while enabling stricter production validation policies.
