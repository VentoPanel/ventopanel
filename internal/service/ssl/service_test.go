package ssl

import (
	"context"
	"errors"
	"testing"
	"time"

	serverdomain "github.com/your-org/ventopanel/internal/domain/server"
	sitedomain "github.com/your-org/ventopanel/internal/domain/site"
)

type siteRepositoryStub struct {
	site    *sitedomain.Site
	updated []string
	err     error
}

func (r *siteRepositoryStub) GetByID(ctx context.Context, id string) (*sitedomain.Site, error) {
	_ = ctx
	_ = id
	if r.site == nil {
		return nil, sitedomain.ErrNotFound
	}
	copy := *r.site
	return &copy, r.err
}

func (r *siteRepositoryStub) Update(ctx context.Context, site *sitedomain.Site) error {
	_ = ctx
	r.updated = append(r.updated, site.Status)
	r.site = site
	return r.err
}

type serverRepositoryStub struct {
	server *serverdomain.Server
	err    error
}

func (r *serverRepositoryStub) GetByID(ctx context.Context, id string) (*serverdomain.Server, error) {
	_ = ctx
	_ = id
	if r.server == nil {
		return nil, serverdomain.ErrNotFound
	}
	copy := *r.server
	return &copy, r.err
}

func (r *serverRepositoryStub) List(ctx context.Context) ([]serverdomain.Server, error) {
	_ = ctx
	if r.server == nil {
		return []serverdomain.Server{}, r.err
	}
	return []serverdomain.Server{*r.server}, r.err
}

func (r *serverRepositoryStub) Update(ctx context.Context, server *serverdomain.Server) error {
	_ = ctx
	r.server = server
	return r.err
}

type sslStub struct {
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

func (s *sslStub) IssueCertificate(ctx context.Context, server serverdomain.Server, domain string) error {
	_ = ctx
	_ = server
	_ = domain
	return s.err
}

func (s *sslStub) RenewCertificates(ctx context.Context, server serverdomain.Server) error {
	_ = ctx
	_ = server
	return s.err
}

func TestExecuteIssueSetsDeployedOnSuccess(t *testing.T) {
	siteRepo := &siteRepositoryStub{
		site: &sitedomain.Site{
			ID:       "site-1",
			ServerID: "srv-1",
			Domain:   "example.com",
			Status:   "ssl_pending",
		},
	}
	serverRepo := &serverRepositoryStub{
		server: &serverdomain.Server{ID: "srv-1"},
	}
	service := NewService(siteRepo, serverRepo, &sslStub{}, nil, &lockStub{}, nil)

	if err := service.ExecuteIssue(context.Background(), IssueSSLPayload{SiteID: "site-1"}); err != nil {
		t.Fatalf("ExecuteIssue returned error: %v", err)
	}

	if len(siteRepo.updated) != 1 || siteRepo.updated[0] != "deployed" {
		t.Fatalf("unexpected status transitions: %#v", siteRepo.updated)
	}
}

func TestExecuteIssueKeepsPendingOnFailure(t *testing.T) {
	siteRepo := &siteRepositoryStub{
		site: &sitedomain.Site{
			ID:       "site-1",
			ServerID: "srv-1",
			Domain:   "example.com",
			Status:   "ssl_pending",
		},
	}
	serverRepo := &serverRepositoryStub{
		server: &serverdomain.Server{ID: "srv-1"},
	}
	service := NewService(siteRepo, serverRepo, &sslStub{err: errors.New("certbot failed")}, nil, &lockStub{}, nil)

	if err := service.ExecuteIssue(context.Background(), IssueSSLPayload{SiteID: "site-1"}); err == nil {
		t.Fatal("expected error, got nil")
	}

	if len(siteRepo.updated) != 1 || siteRepo.updated[0] != "ssl_pending" {
		t.Fatalf("unexpected status transitions: %#v", siteRepo.updated)
	}
}

func TestExecuteRenewCallsSSLManager(t *testing.T) {
	serverRepo := &serverRepositoryStub{
		server: &serverdomain.Server{ID: "srv-1"},
	}
	service := NewService(&siteRepositoryStub{}, serverRepo, &sslStub{}, nil, &lockStub{}, nil)

	if err := service.ExecuteRenew(context.Background(), RenewSSLPayload{ServerID: "srv-1"}); err != nil {
		t.Fatalf("ExecuteRenew returned error: %v", err)
	}
}

func TestExecuteRenewReturnsError(t *testing.T) {
	serverRepo := &serverRepositoryStub{
		server: &serverdomain.Server{ID: "srv-1"},
	}
	service := NewService(&siteRepositoryStub{}, serverRepo, &sslStub{err: errors.New("renew failed")}, nil, &lockStub{}, nil)

	if err := service.ExecuteRenew(context.Background(), RenewSSLPayload{ServerID: "srv-1"}); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestEnqueueDailyRenewForAll(t *testing.T) {
	serverRepo := &serverRepositoryStub{
		server: &serverdomain.Server{ID: "srv-1"},
	}
	service := NewService(&siteRepositoryStub{}, serverRepo, &sslStub{}, nil, &lockStub{}, nil)

	err := service.EnqueueDailyRenewForAll(context.Background(), 0)
	if err == nil {
		t.Fatal("expected error because asynq client is nil")
	}
}
