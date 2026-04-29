package postgres

import (
	"context"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/your-org/ventopanel/internal/domain/settings"
)

type SettingsRepository struct {
	db *pgxpool.Pool
}

func NewSettingsRepository(db *pgxpool.Pool) *SettingsRepository {
	return &SettingsRepository{db: db}
}

func (r *SettingsRepository) Get(ctx context.Context, key string) (string, error) {
	var value string
	err := r.db.QueryRow(ctx,
		`SELECT value FROM app_settings WHERE key = $1`, key,
	).Scan(&value)
	return value, err
}

func (r *SettingsRepository) Set(ctx context.Context, key, value string) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO app_settings (key, value, updated_at)
		 VALUES ($1, $2, NOW())
		 ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()`,
		key, value,
	)
	return err
}

func (r *SettingsRepository) GetNotificationConfig(ctx context.Context) (settings.NotificationConfig, error) {
	rows, err := r.db.Query(ctx,
		`SELECT key, value FROM app_settings
		 WHERE key IN ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		settings.KeyTelegramBotToken,
		settings.KeyTelegramChatID,
		settings.KeyWhatsAppWebhookURL,
		settings.KeyUptimeNotifyDown,
		settings.KeyUptimeNotifyRecovery,
		settings.KeyUptimeFailThreshold,
		settings.KeyUptimeRecoveryThreshold,
		settings.KeyDeployNotifySuccess,
		settings.KeyDeployNotifyFailure,
	)
	if err != nil {
		return settings.NotificationConfig{}, err
	}
	defer rows.Close()

	cfg := settings.NotificationConfig{
		UptimeNotifyDown:        true,
		UptimeNotifyRecovery:    true,
		UptimeFailThreshold:     1,
		UptimeRecoveryThreshold: 1,
		DeployNotifySuccess:     false,
		DeployNotifyFailure:     true,
	}
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return cfg, err
		}
		switch k {
		case settings.KeyTelegramBotToken:
			cfg.TelegramBotToken = v
		case settings.KeyTelegramChatID:
			cfg.TelegramChatID = v
		case settings.KeyWhatsAppWebhookURL:
			cfg.WhatsAppWebhookURL = v
		case settings.KeyUptimeNotifyDown:
			cfg.UptimeNotifyDown = settings.ParseBool(v, true)
		case settings.KeyUptimeNotifyRecovery:
			cfg.UptimeNotifyRecovery = settings.ParseBool(v, true)
		case settings.KeyUptimeFailThreshold:
			cfg.UptimeFailThreshold = settings.ParseIntBounded(v, 1, 1, 60)
		case settings.KeyUptimeRecoveryThreshold:
			cfg.UptimeRecoveryThreshold = settings.ParseIntBounded(v, 1, 1, 60)
		case settings.KeyDeployNotifySuccess:
			cfg.DeployNotifySuccess = settings.ParseBool(v, false)
		case settings.KeyDeployNotifyFailure:
			cfg.DeployNotifyFailure = settings.ParseBool(v, true)
		}
	}
	return cfg, rows.Err()
}

func (r *SettingsRepository) SetNotificationConfig(ctx context.Context, cfg settings.NotificationConfig) error {
	pairs := [][2]string{
		{settings.KeyTelegramBotToken, cfg.TelegramBotToken},
		{settings.KeyTelegramChatID, cfg.TelegramChatID},
		{settings.KeyWhatsAppWebhookURL, cfg.WhatsAppWebhookURL},
		{settings.KeyUptimeNotifyDown, formatBool(cfg.UptimeNotifyDown)},
		{settings.KeyUptimeNotifyRecovery, formatBool(cfg.UptimeNotifyRecovery)},
		{settings.KeyUptimeFailThreshold, strconv.Itoa(settings.ClampInt(cfg.UptimeFailThreshold, 1, 60))},
		{settings.KeyUptimeRecoveryThreshold, strconv.Itoa(settings.ClampInt(cfg.UptimeRecoveryThreshold, 1, 60))},
		{settings.KeyDeployNotifySuccess, formatBool(cfg.DeployNotifySuccess)},
		{settings.KeyDeployNotifyFailure, formatBool(cfg.DeployNotifyFailure)},
	}

	// Use a transaction so all keys are written atomically.
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	for _, kv := range pairs {
		if _, err := tx.Exec(ctx, `
			INSERT INTO app_settings (key, value, updated_at)
			VALUES ($1, $2, NOW())
			ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()`,
			kv[0], kv[1],
		); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *SettingsRepository) GetBackupConfig(ctx context.Context) (settings.BackupConfig, error) {
	rows, err := r.db.Query(ctx,
		`SELECT key, value FROM app_settings WHERE key IN ($1, $2, $3)`,
		settings.KeyBackupAutoEnabled,
		settings.KeyBackupRetentionCount,
		settings.KeyBackupNotifySuccess,
	)
	if err != nil {
		return settings.BackupConfig{}, err
	}
	defer rows.Close()

	cfg := settings.BackupConfig{
		AutoEnabled:    true,
		RetentionCount: 7,
		NotifySuccess:  false,
	}
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return cfg, err
		}
		switch k {
		case settings.KeyBackupAutoEnabled:
			cfg.AutoEnabled = settings.ParseBool(v, true)
		case settings.KeyBackupRetentionCount:
			cfg.RetentionCount = settings.ParseIntBounded(v, 7, 1, 30)
		case settings.KeyBackupNotifySuccess:
			cfg.NotifySuccess = settings.ParseBool(v, false)
		}
	}
	return cfg, rows.Err()
}

func (r *SettingsRepository) SetBackupConfig(ctx context.Context, cfg settings.BackupConfig) error {
	pairs := [][2]string{
		{settings.KeyBackupAutoEnabled, formatBool(cfg.AutoEnabled)},
		{settings.KeyBackupRetentionCount, strconv.Itoa(settings.ClampInt(cfg.RetentionCount, 1, 30))},
		{settings.KeyBackupNotifySuccess, formatBool(cfg.NotifySuccess)},
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	for _, kv := range pairs {
		if _, err := tx.Exec(ctx, `
			INSERT INTO app_settings (key, value, updated_at)
			VALUES ($1, $2, NOW())
			ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()`,
			kv[0], kv[1],
		); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func formatBool(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
