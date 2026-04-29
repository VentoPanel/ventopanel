package filemanager

import (
	"context"
	"fmt"

	"github.com/spf13/afero"
	sftpfs "github.com/spf13/afero/sftpfs"

	serverdomain "github.com/your-org/ventopanel/internal/domain/server"
)

// ServerProvider is a minimal interface over the server repository.
// It allows the Factory to resolve server credentials without importing
// the full postgres repository package.
type ServerProvider interface {
	GetByID(ctx context.Context, id string) (*serverdomain.Server, error)
}

// Factory creates *Service instances backed either by the local OS filesystem
// or by a remote server's filesystem over SFTP.
type Factory struct {
	pool      *SSHPool
	servers   ServerProvider
	localRoot string
}

// NewFactory returns a Factory. pool may be nil — GlobalPool is used in that case.
func NewFactory(servers ServerProvider, localRoot string) *Factory {
	return &Factory{
		pool:      GlobalPool,
		servers:   servers,
		localRoot: localRoot,
	}
}

// Local returns a Service that operates on the local filesystem under localRoot.
func (f *Factory) Local() *Service {
	return NewService(f.localRoot)
}

// ForServer returns a Service that operates on the remote server's filesystem
// over SFTP. The root is "/" (full remote filesystem, path-jailed per request).
// Compress and Extract operations run native zip/unzip commands over SSH.
func (f *Factory) ForServer(ctx context.Context, serverID string) (*Service, error) {
	srv, err := f.servers.GetByID(ctx, serverID)
	if err != nil {
		return nil, fmt.Errorf("server %s not found: %w", serverID, err)
	}

	cfg := ServerDialConfig{
		Host:     srv.Host,
		Port:     srv.Port,
		User:     srv.SSHUser,
		Password: srv.SSHPassword,
		// SSHKey would go here if the domain model adds it later.
	}

	sftpCli, sshCli, err := f.pool.Get(serverID, cfg)
	if err != nil {
		return nil, err
	}

	// Wrap SFTP client in an afero.Fs, then jail it to "/" (remote root).
	fs := afero.NewBasePathFs(sftpfs.New(sftpCli), "/")
	return NewServiceWithFs(fs, "/", sshCli), nil
}

// Resolve returns either a remote SFTP service (when serverID != "") or the
// local service. This is the one-liner convenience method used by handlers.
func (f *Factory) Resolve(ctx context.Context, serverID string) (*Service, error) {
	if serverID == "" {
		return f.Local(), nil
	}
	return f.ForServer(ctx, serverID)
}
