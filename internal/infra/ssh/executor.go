package ssh

import (
	"context"
	"fmt"
	"net"
	"time"

	gossh "golang.org/x/crypto/ssh"

	domain "github.com/your-org/ventopanel/internal/domain/server"
)

type Executor struct {
	dialTimeout time.Duration
}

func NewExecutor(dialTimeout time.Duration) *Executor {
	return &Executor{dialTimeout: dialTimeout}
}

func (e *Executor) Run(ctx context.Context, server domain.Server, command string) error {
	client, err := e.newClient(ctx, server)
	if err != nil {
		return err
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("create ssh session: %w", err)
	}
	defer session.Close()

	if err := session.Run(command); err != nil {
		return fmt.Errorf("run remote command: %w", err)
	}

	return nil
}

func (e *Executor) RunScript(ctx context.Context, server domain.Server, commands []string) error {
	for _, command := range commands {
		if err := e.Run(ctx, server, command); err != nil {
			return fmt.Errorf("run command %q: %w", command, err)
		}
	}

	return nil
}

func (e *Executor) TestConnection(ctx context.Context, server domain.Server) error {
	client, err := e.newClient(ctx, server)
	if err != nil {
		return err
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("create ssh session: %w", err)
	}
	defer session.Close()

	if err := session.Run("echo connected"); err != nil {
		return fmt.Errorf("run remote command: %w", err)
	}

	return nil
}

func (e *Executor) newClient(ctx context.Context, server domain.Server) (*gossh.Client, error) {
	if server.Host == "" {
		return nil, fmt.Errorf("server host is required")
	}
	if server.SSHUser == "" {
		return nil, fmt.Errorf("ssh user is required")
	}
	if server.SSHPassword == "" {
		return nil, fmt.Errorf("ssh password is required")
	}

	port := server.Port
	if port == 0 {
		port = 22
	}

	target := net.JoinHostPort(server.Host, fmt.Sprintf("%d", port))
	dialer := net.Dialer{Timeout: e.dialTimeout}

	conn, err := dialer.DialContext(ctx, "tcp", target)
	if err != nil {
		return nil, fmt.Errorf("dial ssh target: %w", err)
	}

	config := &gossh.ClientConfig{
		User:            server.SSHUser,
		Auth:            []gossh.AuthMethod{gossh.Password(server.SSHPassword)},
		HostKeyCallback: gossh.InsecureIgnoreHostKey(),
		Timeout:         e.dialTimeout,
	}

	clientConn, chans, reqs, err := gossh.NewClientConn(conn, target, config)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("ssh handshake failed: %w", err)
	}

	return gossh.NewClient(clientConn, chans, reqs), nil
}
