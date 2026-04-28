## v0.1.13 — Site Detail Page

### Highlights

- **New `/sites/[id]` page**:
  - Info cards: Runtime, Domain (with direct link to the site), Repository URL, Created date.
  - Status badge with colour-coded variants.
  - **Action buttons**: Deploy, Edit (opens modal form), Delete (with confirmation dialog).
  - **Audit timeline** — full event history for this site: status transitions with colour-coded badges, reasons, and timestamps.
  - "Load older events" cursor-based pagination.
  - Auto-refresh every **15 seconds** with `RefreshIndicator`.
- Site names in the `/sites` table are now clickable links → detail page.
