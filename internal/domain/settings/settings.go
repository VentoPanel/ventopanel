package settings

import "context"

const (
	KeyTelegramBotToken    = "telegram_bot_token"
	KeyTelegramChatID      = "telegram_chat_id"
	KeyWhatsAppWebhookURL  = "whatsapp_webhook_url"
)

// NotificationConfig holds all notification credentials.
type NotificationConfig struct {
	TelegramBotToken   string
	TelegramChatID     string
	WhatsAppWebhookURL string
}

// Repository persists and retrieves application settings.
type Repository interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string) error
	GetNotificationConfig(ctx context.Context) (NotificationConfig, error)
	SetNotificationConfig(ctx context.Context, cfg NotificationConfig) error
}
