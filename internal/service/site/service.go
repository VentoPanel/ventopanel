package site

import (
	"context"
	"strings"

	serverdomain "github.com/your-org/ventopanel/internal/domain/server"
	domain "github.com/your-org/ventopanel/internal/domain/site"
)

type Service struct {
	siteRepo   domain.Repository
	serverRepo domain.ServerRepository
}

func NewService(siteRepo domain.Repository, serverRepo domain.ServerRepository) *Service {
	return &Service{
		siteRepo:   siteRepo,
		serverRepo: serverRepo,
	}
}

func (s *Service) Create(ctx context.Context, input domain.Site) (*domain.Site, error) {
	site := &domain.Site{
		ServerID:      strings.TrimSpace(input.ServerID),
		Name:          strings.TrimSpace(input.Name),
		Domain:        strings.TrimSpace(input.Domain),
		Runtime:       strings.TrimSpace(input.Runtime),
		RepositoryURL: strings.TrimSpace(input.RepositoryURL),
		Status:        strings.TrimSpace(input.Status),
	}

	if site.Status == "" {
		site.Status = "draft"
	}

	if _, err := s.serverRepo.GetByID(ctx, site.ServerID); err != nil {
		if err == serverdomain.ErrNotFound {
			return nil, domain.ErrServerNotFound
		}

		return nil, err
	}

	if err := s.siteRepo.Create(ctx, site); err != nil {
		return nil, err
	}

	return site, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (*domain.Site, error) {
	return s.siteRepo.GetByID(ctx, strings.TrimSpace(id))
}

func (s *Service) List(ctx context.Context) ([]domain.Site, error) {
	return s.siteRepo.List(ctx)
}

func (s *Service) Update(ctx context.Context, input domain.Site) (*domain.Site, error) {
	site := &domain.Site{
		ID:            strings.TrimSpace(input.ID),
		ServerID:      strings.TrimSpace(input.ServerID),
		Name:          strings.TrimSpace(input.Name),
		Domain:        strings.TrimSpace(input.Domain),
		Runtime:       strings.TrimSpace(input.Runtime),
		RepositoryURL: strings.TrimSpace(input.RepositoryURL),
		Status:        strings.TrimSpace(input.Status),
	}

	if site.Status == "" {
		site.Status = "draft"
	}

	if _, err := s.serverRepo.GetByID(ctx, site.ServerID); err != nil {
		if err == serverdomain.ErrNotFound {
			return nil, domain.ErrServerNotFound
		}

		return nil, err
	}

	if err := s.siteRepo.Update(ctx, site); err != nil {
		return nil, err
	}

	return s.siteRepo.GetByID(ctx, site.ID)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.siteRepo.Delete(ctx, strings.TrimSpace(id))
}
