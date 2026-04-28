package team

import domain "github.com/your-org/ventopanel/internal/domain/team"

type Service struct {
	repo domain.Repository
}

func NewService(repo domain.Repository) *Service {
	return &Service{repo: repo}
}
