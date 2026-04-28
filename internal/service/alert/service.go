package alert

import (
	"context"
	"errors"

	domain "github.com/your-org/ventopanel/internal/domain/alert"
	settingsdomain "github.com/your-org/ventopanel/internal/domain/settings"
	"github.com/your-org/ventopanel/internal/infra/notifier"
)

type Service struct {
	// staticNotifiers are used when settingsRepo is nil (legacy/test path).
	staticNotifiers []domain.Notifier
	settingsRepo    settingsdomain.Repository
}

// NewService accepts any number of static notifiers. Nils are skipped.
// Call WithSettingsRepo to enable dynamic config from the database.
func NewService(notifiers ...domain.Notifier) *Service {
	active := make([]domain.Notifier, 0, len(notifiers))
	for _, n := range notifiers {
		if n != nil {
			active = append(active, n)
		}
	}
	return &Service{staticNotifiers: active}
}

// WithSettingsRepo configures the service to build notifiers dynamically from
// the database on every NotifyAll call. Static notifiers are ignored when set.
func (s *Service) WithSettingsRepo(repo settingsdomain.Repository) *Service {
	s.settingsRepo = repo
	return s
}

// NotifyAll sends message to all configured notifiers.
// Errors are collected and joined so all notifiers are always attempted.
func (s *Service) NotifyAll(ctx context.Context, message string) error {
	var notifiers []domain.Notifier

	if s.settingsRepo != nil {
		cfg, err := s.settingsRepo.GetNotificationConfig(ctx)
		if err == nil {
			if cfg.TelegramBotToken != "" && cfg.TelegramChatID != "" {
				notifiers = append(notifiers, notifier.NewTelegram(cfg.TelegramBotToken, cfg.TelegramChatID))
			}
			if cfg.WhatsAppWebhookURL != "" {
				notifiers = append(notifiers, notifier.NewWhatsApp(cfg.WhatsAppWebhookURL))
			}
		}
	} else {
		notifiers = s.staticNotifiers
	}

	var errs []error
	for _, n := range notifiers {
		if err := n.Notify(ctx, message); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
