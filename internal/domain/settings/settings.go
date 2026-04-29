package settings

import "context"

const (
	KeyTelegramBotToken        = "telegram_bot_token"
	KeyTelegramChatID          = "telegram_chat_id"
	KeyWhatsAppWebhookURL      = "whatsapp_webhook_url"
	KeyUptimeNotifyDown        = "uptime_notify_down"
	KeyUptimeNotifyRecovery    = "uptime_notify_recovery"
	KeyUptimeFailThreshold     = "uptime_fail_threshold"
	KeyUptimeRecoveryThreshold = "uptime_recovery_threshold"
	KeyDeployNotifySuccess     = "deploy_notify_success"
	KeyDeployNotifyFailure     = "deploy_notify_failure"
	// Backup schedule settings
	KeyBackupAutoEnabled    = "backup_auto_enabled"
	KeyBackupRetentionCount = "backup_retention_count"
	KeyBackupNotifySuccess  = "backup_notify_success"
)

// BackupConfig holds scheduled-backup preferences.
type BackupConfig struct {
	AutoEnabled    bool
	RetentionCount int  // number of archives to keep (default 7)
	NotifySuccess  bool // send notification on successful backup
}

// NotificationConfig holds all notification credentials.
type NotificationConfig struct {
	TelegramBotToken   string
	TelegramChatID     string
	WhatsAppWebhookURL string
	// Uptime alerts — Telegram/WhatsApp payload respects NotifyDown / NotifyRecovery
	// and consecutive thresholds (anti-flapping).
	UptimeNotifyDown        bool
	UptimeNotifyRecovery    bool
	UptimeFailThreshold     int // consecutive failed checks before DOWN alert (≥1)
	UptimeRecoveryThreshold int // consecutive OK checks before RECOVERY alert (≥1)
	// Deploy alerts
	DeployNotifySuccess bool
	DeployNotifyFailure bool
}

// Repository persists and retrieves application settings.
type Repository interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string) error
	GetNotificationConfig(ctx context.Context) (NotificationConfig, error)
	SetNotificationConfig(ctx context.Context, cfg NotificationConfig) error
	GetBackupConfig(ctx context.Context) (BackupConfig, error)
	SetBackupConfig(ctx context.Context, cfg BackupConfig) error
}
