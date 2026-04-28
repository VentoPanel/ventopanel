## v0.1.12 — Server Monitoring

### Highlights

- **New API endpoint** `GET /api/v1/servers/:id/stats` — runs SSH commands on the remote server and returns live resource metrics:
  - CPU cores + load average (1 min)
  - RAM total / used (MB)
  - Disk total / used / free / used % for `/`
  - System uptime
- **Server detail page** `/servers/[id]` in the frontend:
  - Clickable server name in the table → detail page
  - Info section: provider, SSH user, creation date
  - Monitoring cards with color-coded progress bars (green → yellow → red)
  - Auto-refresh every **30 seconds** with live indicator
  - Connect & Provision buttons directly on the page
