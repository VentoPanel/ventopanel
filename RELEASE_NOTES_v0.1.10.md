## v0.1.10 — Telegram & WhatsApp Alerts

### Highlights

- **Real Telegram notifications** via Bot API (`sendMessage`, HTML parse mode).
- **WhatsApp webhook** support for any compatible API (Evolution, Twilio).
- **Success + failure alerts** for every async task:
  - ✅ Site deployed / 🚨 Site deploy FAILED
  - ✅ Server provisioned / 🚨 Server provisioning FAILED
  - 🔒 SSL issued/renewed / 🚨 SSL issue/renewal FAILED
- Messages include resource IDs and error details for quick triage.
- Fan-out: if one channel fails, the others still receive the alert.

### Configuration

Add to `.env`:

```env
TELEGRAM_BOT_TOKEN=<your-bot-token>
TELEGRAM_CHAT_ID=<your-chat-id>
WHATSAPP_WEBHOOK_URL=<optional>
```

Both variables are optional — omitting them disables the respective channel silently.
