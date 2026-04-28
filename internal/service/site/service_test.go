package site

import (
	"context"
	"testing"

	serverdomain "github.com/your-org/ventopanel/internal/domain/server"
	domain "github.com/your-org/ventopanel/internal/domain/site"
)

type siteRepositoryStub struct {
	created *domain.Site
	updated *domain.Site
	deleted string
	getByID *domain.Site
	err     error
}

func (r *siteRepositoryStub) Create(ctx context.Context, site *domain.Site) error {
	_ = ctx
	site.ID = "site-1"
	r.created = site
	return r.err
}

func (r *siteRepositoryStub) GetByID(ctx context.Context, id string) (*domain.Site, error) {
	_ = ctx
	_ = id
	return r.getByID, r.err
}

func (r *siteRepositoryStub) List(ctx context.Context) ([]domain.Site, error) {
	_ = ctx
	return nil, r.err
}

func (r *siteRepositoryStub) Update(ctx context.Context, site *domain.Site) error {
	_ = ctx
	r.updated = site
	return r.err
}

func (r *siteRepositoryStub) Delete(ctx context.Context, id string) error {
	_ = ctx
	r.deleted = id
	return r.err
}

type serverRepositoryStub struct {
	server *serverdomain.Server
	err    error
}

func (r *serverRepositoryStub) GetByID(ctx context.Context, id string) (*serverdomain.Server, error) {
	_ = ctx
	_ = id
	return r.server, r.err
}

func TestCreateReturnsServerNotFound(t *testing.T) {
	service := NewService(
		&siteRepositoryStub{},
		&serverRepositoryStub{err: serverdomain.ErrNotFound},
	)

	_, err := service.Create(context.Background(), domain.Site{
		ServerID: "missing-server",
		Name:     "landing",
		Domain:   "example.com",
		Runtime:  "php",
	})
	if err != domain.ErrServerNotFound {
		t.Fatalf("expected ErrServerNotFound, got %v", err)
	}
}

func TestCreateSetsDefaultStatus(t *testing.T) {
	siteRepo := &siteRepositoryStub{}
	service := NewService(
		siteRepo,
		&serverRepositoryStub{server: &serverdomain.Server{ID: "srv-1"}},
	)

	site, err := service.Create(context.Background(), domain.Site{
		ServerID: "srv-1",
		Name:     "landing",
		Domain:   "example.com",
		Runtime:  "node",
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	if site.ID != "site-1" {
		t.Fatalf("unexpected id: %s", site.ID)
	}

	if siteRepo.created.Status != "draft" {
		t.Fatalf("expected draft status, got %s", siteRepo.created.Status)
	}
}

func TestUpdateReturnsServerNotFound(t *testing.T) {
	service := NewService(
		&siteRepositoryStub{},
		&serverRepositoryStub{err: serverdomain.ErrNotFound},
	)

	_, err := service.Update(context.Background(), domain.Site{
		ID:       "site-1",
		ServerID: "missing-server",
		Name:     "landing",
		Domain:   "example.com",
		Runtime:  "node",
	})
	if err != domain.ErrServerNotFound {
		t.Fatalf("expected ErrServerNotFound, got %v", err)
	}
}

func TestUpdateSetsDefaultStatus(t *testing.T) {
	siteRepo := &siteRepositoryStub{
		getByID: &domain.Site{
			ID:       "site-1",
			ServerID: "srv-1",
			Name:     "landing",
			Domain:   "example.com",
			Runtime:  "node",
			Status:   "draft",
		},
	}
	service := NewService(
		siteRepo,
		&serverRepositoryStub{server: &serverdomain.Server{ID: "srv-1"}},
	)

	site, err := service.Update(context.Background(), domain.Site{
		ID:       "site-1",
		ServerID: "srv-1",
		Name:     " landing ",
		Domain:   " example.com ",
		Runtime:  " node ",
	})
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}

	if siteRepo.updated == nil || siteRepo.updated.Status != "draft" {
		t.Fatalf("expected default draft status, got %#v", siteRepo.updated)
	}

	if site.ID != "site-1" {
		t.Fatalf("unexpected updated site: %#v", site)
	}
}

func TestDeletePassesTrimmedID(t *testing.T) {
	siteRepo := &siteRepositoryStub{}
	service := NewService(
		siteRepo,
		&serverRepositoryStub{},
	)

	if err := service.Delete(context.Background(), " site-1 "); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}

	if siteRepo.deleted != "site-1" {
		t.Fatalf("expected trimmed id, got %q", siteRepo.deleted)
	}
}
