package config

import (
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	AppName            string        `env:"APP_NAME" env-default:"VentoPanel"`
	AppEnv             string        `env:"APP_ENV" env-default:"development"`
	HTTPPort           string        `env:"HTTP_PORT" env-default:"8080"`
	LogLevel           string        `env:"LOG_LEVEL" env-default:"info"`
	PostgresDSN        string        `env:"POSTGRES_DSN" env-required:"true"`
	RedisAddr          string        `env:"REDIS_ADDR" env-default:"localhost:6379"`
	RedisDB            int           `env:"REDIS_DB" env-default:"0"`
	SSHDialTimeout     time.Duration `env:"SSH_DIAL_TIMEOUT" env-default:"10s"`
	AppEncryptionKey   string        `env:"APP_ENCRYPTION_KEY" env-default:"0123456789abcdef0123456789abcdef"`
	SSLCertbotEmail    string        `env:"SSL_CERTBOT_EMAIL" env-default:""`
	TelegramBotToken   string        `env:"TELEGRAM_BOT_TOKEN"`
	TelegramChatID     string        `env:"TELEGRAM_CHAT_ID"`
	WhatsAppWebhookURL string        `env:"WHATSAPP_WEBHOOK_URL"`
	AuthJWTSecret      string        `env:"AUTH_JWT_SECRET" env-default:"dev-insecure-change-me"`
	AuthAllowHeaders   bool          `env:"AUTH_ALLOW_HEADER_FALLBACK" env-default:"false"`
	AuthJWTIssuer      string        `env:"AUTH_JWT_ISSUER" env-default:""`
	AuthJWTAudience    string        `env:"AUTH_JWT_AUDIENCE" env-default:""`
	BackupDir          string        `env:"BACKUP_DIR" env-default:"/data/backups"`
	BackupKeepCount    int           `env:"BACKUP_KEEP_COUNT" env-default:"7"`
	FileManagerRoot    string        `env:"FILE_MANAGER_ROOT" env-default:"/var/www"`
}

func Load() (*Config, error) {
	var cfg Config
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
