package server

import (
	"context"
	"errors"
	"testing"

	domain "github.com/your-org/ventopanel/internal/domain/server"
)

type repositoryStub struct {
	created *domain.Server
	list    []domain.Server
	getByID *domain.Server
	updated *domain.Server
	deleted string
	err     error
}

func (r *repositoryStub) Ping(ctx context.Context) error {
	_ = ctx
	return r.err
}

func (r *repositoryStub) Create(ctx context.Context, server *domain.Server) error {
	_ = ctx
	server.ID = "srv-1"
	r.created = server
	return r.err
}

func (r *repositoryStub) GetByID(ctx context.Context, id string) (*domain.Server, error) {
	_ = ctx
	_ = id
	return r.getByID, r.err
}

func (r *repositoryStub) List(ctx context.Context) ([]domain.Server, error) {
	_ = ctx
	return r.list, r.err
}

func (r *repositoryStub) Update(ctx context.Context, server *domain.Server) error {
	_ = ctx
	r.updated = server
	return r.err
}

func (r *repositoryStub) Delete(ctx context.Context, id string) error {
	_ = ctx
	r.deleted = id
	return r.err
}

type sshStub struct{}

func (s *sshStub) Run(ctx context.Context, host domain.Server, command string) error {
	_ = ctx
	_ = host
	_ = command
	return nil
}

func (s *sshStub) RunScript(ctx context.Context, host domain.Server, commands []string) error {
	_ = ctx
	_ = host
	_ = commands
	return nil
}

func (s *sshStub) TestConnection(ctx context.Context, server domain.Server) error {
	_ = ctx
	_ = server
	return nil
}

type failingSSHStub struct {
	err error
}

func (s *failingSSHStub) Run(ctx context.Context, host domain.Server, command string) error {
	_ = ctx
	_ = host
	_ = command
	return s.err
}

func (s *failingSSHStub) RunScript(ctx context.Context, host domain.Server, commands []string) error {
	_ = ctx
	_ = host
	_ = commands
	return s.err
}

func (s *failingSSHStub) TestConnection(ctx context.Context, server domain.Server) error {
	_ = ctx
	_ = server
	return s.err
}

func TestCreateSetsDefaults(t *testing.T) {
	repo := &repositoryStub{}
	service := NewService(repo, &sshStub{}, nil)

	server, err := service.Create(context.Background(), domain.Server{
		Name:     "primary",
		Host:     "10.0.0.1",
		Provider: "hetzner",
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	if server.ID != "srv-1" {
		t.Fatalf("unexpected id: %s", server.ID)
	}

	if repo.created.Port != 22 {
		t.Fatalf("expected default port 22, got %d", repo.created.Port)
	}

	if repo.created.Status != "pending" {
		t.Fatalf("expected default status pending, got %s", repo.created.Status)
	}

	if repo.created.SSHUser != "root" {
		t.Fatalf("expected default ssh user root, got %s", repo.created.SSHUser)
	}
}

func TestUpdateReturnsRefreshedServer(t *testing.T) {
	repo := &repositoryStub{
		getByID: &domain.Server{
			ID:       "srv-1",
			Name:     "updated",
			Host:     "10.0.0.2",
			Port:     22,
			Provider: "do",
			Status:   "ready",
		},
	}
	service := NewService(repo, &sshStub{}, nil)

	server, err := service.Update(context.Background(), domain.Server{
		ID:       "srv-1",
		Name:     " updated ",
		Host:     " 10.0.0.2 ",
		Provider: " do ",
		Status:   " ready ",
	})
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}

	if repo.updated == nil || repo.updated.Name != "updated" || repo.updated.Host != "10.0.0.2" {
		t.Fatalf("update payload was not normalized: %#v", repo.updated)
	}

	if server.ID != "srv-1" || server.Status != "ready" {
		t.Fatalf("unexpected updated server: %#v", server)
	}
}

func TestDeletePassesTrimmedID(t *testing.T) {
	repo := &repositoryStub{}
	service := NewService(repo, &sshStub{}, nil)

	if err := service.Delete(context.Background(), " srv-1 "); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}

	if repo.deleted != "srv-1" {
		t.Fatalf("expected trimmed id, got %q", repo.deleted)
	}
}

func TestConnectMarksServerConnected(t *testing.T) {
	repo := &repositoryStub{
		getByID: &domain.Server{
			ID:          "srv-1",
			Name:        "primary",
			Host:        "10.0.0.1",
			Port:        22,
			Provider:    "hetzner",
			Status:      "pending",
			SSHUser:     "root",
			SSHPassword: "secret",
		},
	}
	service := NewService(repo, &sshStub{}, nil)

	server, err := service.Connect(context.Background(), "srv-1")
	if err != nil {
		t.Fatalf("Connect returned error: %v", err)
	}

	if repo.updated == nil || repo.updated.Status != "connected" {
		t.Fatalf("expected connected status update, got %#v", repo.updated)
	}

	if server == nil || server.ID != "srv-1" {
		t.Fatalf("unexpected server: %#v", server)
	}
}

func TestConnectMarksServerFailedOnSSHError(t *testing.T) {
	repo := &repositoryStub{
		getByID: &domain.Server{
			ID:          "srv-1",
			Name:        "primary",
			Host:        "10.0.0.1",
			Port:        22,
			Provider:    "hetzner",
			Status:      "pending",
			SSHUser:     "root",
			SSHPassword: "secret",
		},
	}
	service := NewService(repo, &failingSSHStub{err: errors.New("dial failed")}, nil)

	server, err := service.Connect(context.Background(), "srv-1")
	if err == nil {
		t.Fatal("expected connect error, got nil")
	}

	if server != nil {
		t.Fatalf("expected nil server on failure, got %#v", server)
	}

	if repo.updated == nil || repo.updated.Status != "connection_failed" {
		t.Fatalf("expected connection_failed status update, got %#v", repo.updated)
	}
}
