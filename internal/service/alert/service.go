package alert

import (
	"context"
	"errors"

	domain "github.com/your-org/ventopanel/internal/domain/alert"
)

type Service struct {
	notifiers []domain.Notifier
}

// NewService accepts any number of notifiers. Nils are skipped.
func NewService(notifiers ...domain.Notifier) *Service {
	active := make([]domain.Notifier, 0, len(notifiers))
	for _, n := range notifiers {
		if n != nil {
			active = append(active, n)
		}
	}
	return &Service{notifiers: active}
}

// NotifyAll sends message to all configured notifiers.
// Errors are collected and joined so all notifiers are always attempted.
func (s *Service) NotifyAll(ctx context.Context, message string) error {
	var errs []error
	for _, n := range s.notifiers {
		if err := n.Notify(ctx, message); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
