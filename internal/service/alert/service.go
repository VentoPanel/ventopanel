package alert

import (
	"context"

	domain "github.com/your-org/ventopanel/internal/domain/alert"
)

type Service struct {
	telegram domain.Notifier
	whatsApp domain.Notifier
}

func NewService(telegram domain.Notifier, whatsApp domain.Notifier) *Service {
	return &Service{
		telegram: telegram,
		whatsApp: whatsApp,
	}
}

func (s *Service) NotifyAll(ctx context.Context, message string) error {
	if err := s.telegram.Notify(ctx, message); err != nil {
		return err
	}

	if err := s.whatsApp.Notify(ctx, message); err != nil {
		return err
	}

	return nil
}
