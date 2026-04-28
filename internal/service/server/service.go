package server

import (
	"context"
	"strings"

	auditdomain "github.com/your-org/ventopanel/internal/domain/audit"
	"github.com/your-org/ventopanel/internal/domain/lifecycle"
	domain "github.com/your-org/ventopanel/internal/domain/server"
)

type Service struct {
	repo        domain.Repository
	sshExecutor domain.SSHExecutor
	audit       auditdomain.StatusEventWriter
}

func NewService(repo domain.Repository, sshExecutor domain.SSHExecutor, audit auditdomain.StatusEventWriter) *Service {
	return &Service{repo: repo, sshExecutor: sshExecutor, audit: audit}
}

func (s *Service) Health(ctx context.Context) error {
	return s.repo.Ping(ctx)
}

func (s *Service) Create(ctx context.Context, input domain.Server) (*domain.Server, error) {
	server := &domain.Server{
		Name:        strings.TrimSpace(input.Name),
		Host:        strings.TrimSpace(input.Host),
		Port:        input.Port,
		Provider:    strings.TrimSpace(input.Provider),
		Status:      strings.TrimSpace(input.Status),
		SSHUser:     strings.TrimSpace(input.SSHUser),
		SSHPassword: strings.TrimSpace(input.SSHPassword),
	}

	if server.Port == 0 {
		server.Port = 22
	}

	if server.Status == "" {
		server.Status = "pending"
	}

	if server.SSHUser == "" {
		server.SSHUser = "root"
	}

	if strings.TrimSpace(server.LastRenewStatus) == "" {
		server.LastRenewStatus = "unknown"
	}

	if err := s.repo.Create(ctx, server); err != nil {
		return nil, err
	}

	return server, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (*domain.Server, error) {
	return s.repo.GetByID(ctx, strings.TrimSpace(id))
}

func (s *Service) List(ctx context.Context) ([]domain.Server, error) {
	return s.repo.List(ctx)
}

func (s *Service) Update(ctx context.Context, input domain.Server) (*domain.Server, error) {
	server := &domain.Server{
		ID:          strings.TrimSpace(input.ID),
		Name:        strings.TrimSpace(input.Name),
		Host:        strings.TrimSpace(input.Host),
		Port:        input.Port,
		Provider:    strings.TrimSpace(input.Provider),
		Status:      strings.TrimSpace(input.Status),
		SSHUser:     strings.TrimSpace(input.SSHUser),
		SSHPassword: strings.TrimSpace(input.SSHPassword),
	}

	if server.Port == 0 {
		server.Port = 22
	}

	if server.Status == "" {
		server.Status = "pending"
	}

	if server.SSHUser == "" {
		server.SSHUser = "root"
	}

	if strings.TrimSpace(server.LastRenewStatus) == "" {
		server.LastRenewStatus = "unknown"
	}

	if err := s.repo.Update(ctx, server); err != nil {
		return nil, err
	}

	return s.repo.GetByID(ctx, server.ID)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, strings.TrimSpace(id))
}

func (s *Service) Connect(ctx context.Context, id string) (*domain.Server, error) {
	server, err := s.repo.GetByID(ctx, strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}

	if err := s.sshExecutor.TestConnection(ctx, *server); err != nil {
		if transitionErr := lifecycle.EnsureServerTransition(server.Status, "connection_failed"); transitionErr != nil {
			return nil, transitionErr
		}
		prev := server.Status
		server.Status = "connection_failed"
		if updateErr := s.repo.Update(ctx, server); updateErr != nil {
			return nil, updateErr
		}
		s.writeAudit("server", server.ID, prev, server.Status, "ssh_connect_failed", "connect")

		return nil, err
	}

	if err := lifecycle.EnsureServerTransition(server.Status, "connected"); err != nil {
		return nil, err
	}
	prev := server.Status
	server.Status = "connected"
	if err := s.repo.Update(ctx, server); err != nil {
		return nil, err
	}
	s.writeAudit("server", server.ID, prev, server.Status, "ssh_connect_success", "connect")

	return s.repo.GetByID(ctx, server.ID)
}

func (s *Service) writeAudit(resourceType, resourceID, from, to, reason, taskID string) {
	if s.audit == nil || from == to {
		return
	}
	_ = s.audit.WriteStatusEvent(auditdomain.StatusEvent{
		ResourceType: resourceType,
		ResourceID:   resourceID,
		FromStatus:   from,
		ToStatus:     to,
		Reason:       reason,
		TaskID:       taskID,
	})
}
