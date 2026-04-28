package provision

import (
	"context"
	"errors"
	"testing"
	"time"

	domain "github.com/your-org/ventopanel/internal/domain/server"
)

type repositoryStub struct {
	server  *domain.Server
	updated []string
	err     error
}

func (r *repositoryStub) Ping(ctx context.Context) error {
	_ = ctx
	return r.err
}

func (r *repositoryStub) Create(ctx context.Context, server *domain.Server) error {
	_ = ctx
	_ = server
	return r.err
}

func (r *repositoryStub) GetByID(ctx context.Context, id string) (*domain.Server, error) {
	_ = ctx
	_ = id
	if r.server == nil {
		return nil, domain.ErrNotFound
	}
	copy := *r.server
	return &copy, r.err
}

func (r *repositoryStub) List(ctx context.Context) ([]domain.Server, error) {
	_ = ctx
	return nil, r.err
}

func (r *repositoryStub) Update(ctx context.Context, server *domain.Server) error {
	_ = ctx
	r.updated = append(r.updated, server.Status)
	r.server = server
	return r.err
}

func (r *repositoryStub) Delete(ctx context.Context, id string) error {
	_ = ctx
	_ = id
	return r.err
}

type sshStub struct {
	err error
}

type lockStub struct{}

func (l *lockStub) Acquire(ctx context.Context, key string, ttl time.Duration) error {
	_ = ctx
	_ = key
	_ = ttl
	return nil
}

func (l *lockStub) Release(ctx context.Context, key string) error {
	_ = ctx
	_ = key
	return nil
}

func (s *sshStub) Run(ctx context.Context, server domain.Server, command string) error {
	_ = ctx
	_ = server
	_ = command
	return s.err
}

func (s *sshStub) RunScript(ctx context.Context, server domain.Server, commands []string) error {
	_ = ctx
	_ = server
	_ = commands
	return s.err
}

func (s *sshStub) TestConnection(ctx context.Context, server domain.Server) error {
	_ = ctx
	_ = server
	return s.err
}

func TestExecuteProvisionMarksReady(t *testing.T) {
	repo := &repositoryStub{
		server: &domain.Server{
			ID:          "srv-1",
			Host:        "10.0.0.1",
			Port:        22,
			SSHUser:     "root",
			SSHPassword: "secret",
			Status:      "connected",
		},
	}
	service := NewService(repo, &sshStub{}, nil, &lockStub{}, nil)

	err := service.ExecuteProvision(context.Background(), ProvisionServerPayload{ServerID: "srv-1"})
	if err != nil {
		t.Fatalf("ExecuteProvision returned error: %v", err)
	}

	if len(repo.updated) != 2 || repo.updated[0] != "provisioning" || repo.updated[1] != "ready_for_deploy" {
		t.Fatalf("unexpected statuses: %#v", repo.updated)
	}
}

func TestExecuteProvisionMarksFailed(t *testing.T) {
	repo := &repositoryStub{
		server: &domain.Server{
			ID:          "srv-1",
			Host:        "10.0.0.1",
			Port:        22,
			SSHUser:     "root",
			SSHPassword: "secret",
			Status:      "connected",
		},
	}
	service := NewService(repo, &sshStub{err: errors.New("apt failed")}, nil, &lockStub{}, nil)

	err := service.ExecuteProvision(context.Background(), ProvisionServerPayload{ServerID: "srv-1"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if len(repo.updated) != 2 || repo.updated[1] != "provision_failed" {
		t.Fatalf("unexpected statuses: %#v", repo.updated)
	}
}
