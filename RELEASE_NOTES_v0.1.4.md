## v0.1.4

### Highlights

- Added ACL deny metrics for production observability:
  - `ventopanel_acl_denied_total{resource_type,reason}`
- Added audit writes for ACL deny decisions on server/site protected endpoints.
- Added integration test that verifies denied site access creates an `access_denied` status event.

### Notes

- Deny audit writes are best-effort and do not block API responses.
- Denies caused by missing team identity are counted in metrics, while audit write is skipped when resource id is absent.
