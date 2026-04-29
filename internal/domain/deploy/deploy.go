package deploy

import (
	"context"
	"io"

	serverdomain "github.com/your-org/ventopanel/internal/domain/server"
	sitedomain "github.com/your-org/ventopanel/internal/domain/site"
)

type SiteRepository interface {
	GetByID(ctx context.Context, id string) (*sitedomain.Site, error)
	Update(ctx context.Context, site *sitedomain.Site) error
}

type ServerRepository interface {
	GetByID(ctx context.Context, id string) (*serverdomain.Server, error)
	List(ctx context.Context) ([]serverdomain.Server, error)
	Update(ctx context.Context, server *serverdomain.Server) error
}

type SSHExecutor interface {
	Run(ctx context.Context, server serverdomain.Server, command string) error
	RunScript(ctx context.Context, server serverdomain.Server, commands []string) error
	RunOutput(ctx context.Context, server serverdomain.Server, command string) (string, error)
	RunStream(ctx context.Context, server serverdomain.Server, command string, w io.Writer) error
}

type FirewallManager interface {
	EnsureDefaultRules(ctx context.Context, host string) error
}

type SSLManager interface {
	IssueCertificate(ctx context.Context, server serverdomain.Server, domain string) error
	RenewCertificates(ctx context.Context, server serverdomain.Server) error
}
