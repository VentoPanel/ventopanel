package filemanager

import (
	"fmt"
	"sync"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// ─── Pool entry ───────────────────────────────────────────────────────────────

type poolEntry struct {
	mu      sync.Mutex
	sshCli  *ssh.Client
	sftpCli *sftp.Client
}

func (e *poolEntry) isAlive() bool {
	if e.sshCli == nil || e.sftpCli == nil {
		return false
	}
	// Cheap health check: ask SFTP for current working directory.
	_, err := e.sftpCli.Getwd()
	return err == nil
}

func (e *poolEntry) close() {
	if e.sftpCli != nil {
		e.sftpCli.Close()
		e.sftpCli = nil
	}
	if e.sshCli != nil {
		e.sshCli.Close()
		e.sshCli = nil
	}
}

// ─── Pool ─────────────────────────────────────────────────────────────────────

// SSHPool maintains reusable SSH + SFTP connections keyed by server ID.
// Each server gets at most one live connection; stale connections are
// transparently re-established on next access.
type SSHPool struct {
	entries sync.Map // string (serverID) -> *poolEntry
}

// GlobalPool is the application-level singleton used by Factory.
var GlobalPool = &SSHPool{}

// ServerDialConfig holds the credentials needed to open an SSH connection.
type ServerDialConfig struct {
	Host     string
	Port     int
	User     string
	Password string // used when PEMKey is empty
	PEMKey   string // PEM-encoded private key (takes priority over Password)
}

// Get returns a live *sftp.Client and *ssh.Client for the given server,
// creating a new connection if none exists or the existing one is stale.
func (p *SSHPool) Get(serverID string, cfg ServerDialConfig) (*sftp.Client, *ssh.Client, error) {
	// LoadOrStore atomically inserts an empty entry if the key is missing.
	raw, _ := p.entries.LoadOrStore(serverID, &poolEntry{})
	entry := raw.(*poolEntry)

	entry.mu.Lock()
	defer entry.mu.Unlock()

	if entry.isAlive() {
		return entry.sftpCli, entry.sshCli, nil
	}

	// Stale or missing — reconnect.
	entry.close()

	sshCli, err := dialSSH(cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("ssh dial %s: %w", cfg.Host, err)
	}

	sftpCli, err := sftp.NewClient(sshCli)
	if err != nil {
		sshCli.Close()
		return nil, nil, fmt.Errorf("sftp client %s: %w", cfg.Host, err)
	}

	entry.sshCli = sshCli
	entry.sftpCli = sftpCli
	return sftpCli, sshCli, nil
}

// Invalidate closes and removes the connection for serverID.
// Call this when you receive an error mid-operation so the next request
// creates a fresh connection instead of reusing a broken one.
func (p *SSHPool) Invalidate(serverID string) {
	if raw, ok := p.entries.LoadAndDelete(serverID); ok {
		e := raw.(*poolEntry)
		e.mu.Lock()
		e.close()
		e.mu.Unlock()
	}
}

// ─── SSH dialer ───────────────────────────────────────────────────────────────

// DialSSH is the public entry-point for packages that need a raw *ssh.Client
// without going through the pool (e.g. terminal sessions).
func DialSSH(cfg ServerDialConfig) (*ssh.Client, error) { return dialSSH(cfg) }

func dialSSH(cfg ServerDialConfig) (*ssh.Client, error) {
	var authMethods []ssh.AuthMethod

	if cfg.PEMKey != "" {
		signer, err := ssh.ParsePrivateKey([]byte(cfg.PEMKey))
		if err != nil {
			return nil, fmt.Errorf("parse private key: %w", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}

	if cfg.Password != "" {
		authMethods = append(authMethods, ssh.Password(cfg.Password))
		// keyboard-interactive fallback (some servers need it)
		authMethods = append(authMethods, ssh.KeyboardInteractive(
			func(_, _ string, questions []string, _ []bool) ([]string, error) {
				answers := make([]string, len(questions))
				for i := range questions {
					answers[i] = cfg.Password
				}
				return answers, nil
			},
		))
	}

	port := cfg.Port
	if port == 0 {
		port = 22
	}

	clientCfg := &ssh.ClientConfig{
		User:            cfg.User,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //nolint:gosec // intentional for managed servers
		Timeout:         15 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", cfg.Host, port)
	return ssh.Dial("tcp", addr, clientCfg)
}
