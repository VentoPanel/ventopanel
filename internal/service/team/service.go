package team

import "context"
import domain "github.com/your-org/ventopanel/internal/domain/team"

type Service struct {
	repo domain.Repository
}

func NewService(repo domain.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) HasSiteAccess(ctx context.Context, teamID, siteID string) (bool, error) {
	return s.repo.HasSiteAccess(ctx, teamID, siteID)
}

func (s *Service) GetSiteRole(ctx context.Context, teamID, siteID string) (string, error) {
	return s.repo.GetSiteRole(ctx, teamID, siteID)
}

func (s *Service) HasServerAccess(ctx context.Context, teamID, serverID string) (bool, error) {
	return s.repo.HasServerAccess(ctx, teamID, serverID)
}

func (s *Service) GetServerRole(ctx context.Context, teamID, serverID string) (string, error) {
	return s.repo.GetServerRole(ctx, teamID, serverID)
}
