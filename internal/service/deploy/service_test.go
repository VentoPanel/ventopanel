package deploy

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

type sshStub struct {
	err error
}

func (s *sshStub) Run(ctx context.Context, server serverdomain.Server, command string) error {
	_ = ctx
	_ = server
	_ = command
	return s.err
}

func (s *sshStub) RunScript(ctx context.Context, server serverdomain.Server, commands []string) error {
	_ = ctx
	_ = server
	_ = commands
	return s.err
}

type firewallStub struct{}

func (f *firewallStub) EnsureDefaultRules(ctx context.Context, host string) error {
	_ = ctx
	_ = host
	return nil
}

type sslStub struct {
	err error
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

type sslQueueStub struct {
	siteID string
	err    error
}

type lockStub struct {
	err error
}

func (l *lockStub) Acquire(ctx context.Context, key string, ttl time.Duration) error {
	_ = ctx
	_ = key
	_ = ttl
	return l.err
}

func (l *lockStub) Release(ctx context.Context, key string) error {
	_ = ctx
	_ = key
	return nil
}

func (s *sslQueueStub) EnqueueIssue(ctx context.Context, siteID string) error {
	_ = ctx
	s.siteID = siteID
	return s.err
}

func TestExecuteDeploySetsDeployedStatus(t *testing.T) {
	siteRepo := &siteRepositoryStub{
		site: &sitedomain.Site{
			ID:       "site-1",
			ServerID: "srv-1",
			Domain:   "example.com",
			Runtime:  "node",
			Status:   "draft",
		},
	}
	serverRepo := &serverRepositoryStub{
		server: &serverdomain.Server{
			ID:          "srv-1",
			Host:        "10.0.0.1",
			Port:        22,
			SSHUser:     "root",
			SSHPassword: "secret",
		},
	}

	service := NewService(siteRepo, serverRepo, &sshStub{}, &firewallStub{}, &sslStub{}, nil, nil, &lockStub{}, nil)
	err := service.ExecuteDeploy(context.Background(), DeploySitePayload{SiteID: "site-1"})
	if err != nil {
		t.Fatalf("ExecuteDeploy returned error: %v", err)
	}

	if len(siteRepo.updated) != 2 || siteRepo.updated[0] != "deploying" || siteRepo.updated[1] != "deployed" {
		t.Fatalf("unexpected status transitions: %#v", siteRepo.updated)
	}
}

func TestExecuteDeploySetsFailedStatusOnSSHError(t *testing.T) {
	siteRepo := &siteRepositoryStub{
		site: &sitedomain.Site{
			ID:       "site-1",
			ServerID: "srv-1",
			Domain:   "example.com",
			Runtime:  "node",
			Status:   "draft",
		},
	}
	serverRepo := &serverRepositoryStub{
		server: &serverdomain.Server{
			ID:          "srv-1",
			Host:        "10.0.0.1",
			Port:        22,
			SSHUser:     "root",
			SSHPassword: "secret",
		},
	}

	service := NewService(siteRepo, serverRepo, &sshStub{err: errors.New("docker failed")}, &firewallStub{}, &sslStub{}, nil, nil, &lockStub{}, nil)
	err := service.ExecuteDeploy(context.Background(), DeploySitePayload{SiteID: "site-1"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if len(siteRepo.updated) != 2 || siteRepo.updated[1] != "deploy_failed" {
		t.Fatalf("unexpected status transitions: %#v", siteRepo.updated)
	}
}

func TestExecuteDeploySetsSSLPendingOnSSLError(t *testing.T) {
	siteRepo := &siteRepositoryStub{
		site: &sitedomain.Site{
			ID:       "site-1",
			ServerID: "srv-1",
			Domain:   "example.com",
			Runtime:  "node",
			Status:   "draft",
		},
	}
	serverRepo := &serverRepositoryStub{
		server: &serverdomain.Server{
			ID:          "srv-1",
			Host:        "10.0.0.1",
			Port:        22,
			SSHUser:     "root",
			SSHPassword: "secret",
		},
	}

	queue := &sslQueueStub{}
	service := NewService(siteRepo, serverRepo, &sshStub{}, &firewallStub{}, &sslStub{err: errors.New("certbot failed")}, queue, nil, &lockStub{}, nil)
	err := service.ExecuteDeploy(context.Background(), DeploySitePayload{SiteID: "site-1"})
	if err != nil {
		t.Fatalf("expected nil error because ssl is deferred, got %v", err)
	}

	if len(siteRepo.updated) != 2 || siteRepo.updated[1] != "ssl_pending" {
		t.Fatalf("unexpected status transitions: %#v", siteRepo.updated)
	}

	if queue.siteID != "site-1" {
		t.Fatalf("expected ssl queue site id site-1, got %q", queue.siteID)
	}
}
