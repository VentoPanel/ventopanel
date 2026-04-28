package security

import (
	"context"
	"fmt"
	"strings"

	serverdomain "github.com/your-org/ventopanel/internal/domain/server"
)

type FirewallManager struct{}

func NewFirewallManager() *FirewallManager {
	return &FirewallManager{}
}

func (m *FirewallManager) EnsureDefaultRules(ctx context.Context, host string) error {
	_ = ctx
	_ = host
	return nil
}

type sshExecutor interface {
	Run(ctx context.Context, server serverdomain.Server, command string) error
}

type SSLManager struct {
	ssh          sshExecutor
	certbotEmail string
}

func NewSSLManager(ssh sshExecutor, certbotEmail string) *SSLManager {
	return &SSLManager{
		ssh:          ssh,
		certbotEmail: strings.TrimSpace(certbotEmail),
	}
}

func (m *SSLManager) IssueCertificate(ctx context.Context, server serverdomain.Server, domain string) error {
	var command string
	if m.certbotEmail != "" {
		command = fmt.Sprintf(
			"certbot --nginx -d %s --non-interactive --agree-tos --email %s --redirect",
			domain,
			m.certbotEmail,
		)
	} else {
		command = fmt.Sprintf(
			"certbot --nginx -d %s --non-interactive --agree-tos --register-unsafely-without-email --redirect",
			domain,
		)
	}

	return m.ssh.Run(ctx, server, command)
}

func (m *SSLManager) RenewCertificates(ctx context.Context, server serverdomain.Server) error {
	command := "certbot renew --nginx --non-interactive --quiet"
	return m.ssh.Run(ctx, server, command)
}
