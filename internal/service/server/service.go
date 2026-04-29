package server

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	auditdomain "github.com/your-org/ventopanel/internal/domain/audit"
	"github.com/your-org/ventopanel/internal/domain/lifecycle"
	domain "github.com/your-org/ventopanel/internal/domain/server"
)

type Service struct {
	repo        domain.Repository
	sshExecutor domain.SSHExecutor
	audit       auditdomain.StatusEventWriter
}

func NewService(repo domain.Repository, sshExecutor domain.SSHExecutor, audit auditdomain.StatusEventWriter) *Service {
	return &Service{repo: repo, sshExecutor: sshExecutor, audit: audit}
}

func (s *Service) Health(ctx context.Context) error {
	return s.repo.Ping(ctx)
}

func (s *Service) Create(ctx context.Context, input domain.Server) (*domain.Server, error) {
	server := &domain.Server{
		Name:        strings.TrimSpace(input.Name),
		Host:        strings.TrimSpace(input.Host),
		Port:        input.Port,
		Provider:    strings.TrimSpace(input.Provider),
		Status:      strings.TrimSpace(input.Status),
		SSHUser:     strings.TrimSpace(input.SSHUser),
		SSHPassword: strings.TrimSpace(input.SSHPassword),
	}

	if server.Port == 0 {
		server.Port = 22
	}

	if server.Status == "" {
		server.Status = "pending"
	}

	if server.SSHUser == "" {
		server.SSHUser = "root"
	}

	if strings.TrimSpace(server.LastRenewStatus) == "" {
		server.LastRenewStatus = "unknown"
	}

	if err := s.repo.Create(ctx, server); err != nil {
		return nil, err
	}

	return server, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (*domain.Server, error) {
	return s.repo.GetByID(ctx, strings.TrimSpace(id))
}

func (s *Service) List(ctx context.Context) ([]domain.Server, error) {
	return s.repo.List(ctx)
}

func (s *Service) Update(ctx context.Context, input domain.Server) (*domain.Server, error) {
	server := &domain.Server{
		ID:          strings.TrimSpace(input.ID),
		Name:        strings.TrimSpace(input.Name),
		Host:        strings.TrimSpace(input.Host),
		Port:        input.Port,
		Provider:    strings.TrimSpace(input.Provider),
		Status:      strings.TrimSpace(input.Status),
		SSHUser:     strings.TrimSpace(input.SSHUser),
		SSHPassword: strings.TrimSpace(input.SSHPassword),
	}

	if server.Port == 0 {
		server.Port = 22
	}

	if server.Status == "" {
		server.Status = "pending"
	}

	if server.SSHUser == "" {
		server.SSHUser = "root"
	}

	if strings.TrimSpace(server.LastRenewStatus) == "" {
		server.LastRenewStatus = "unknown"
	}

	if err := s.repo.Update(ctx, server); err != nil {
		return nil, err
	}

	return s.repo.GetByID(ctx, server.ID)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, strings.TrimSpace(id))
}

func (s *Service) Connect(ctx context.Context, id string) (*domain.Server, error) {
	server, err := s.repo.GetByID(ctx, strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}

	if err := s.sshExecutor.TestConnection(ctx, *server); err != nil {
		if transitionErr := lifecycle.EnsureServerTransition(server.Status, "connection_failed"); transitionErr != nil {
			return nil, transitionErr
		}
		prev := server.Status
		server.Status = "connection_failed"
		if updateErr := s.repo.Update(ctx, server); updateErr != nil {
			return nil, updateErr
		}
		s.writeAudit("server", server.ID, prev, server.Status, "ssh_connect_failed", "connect")

		return nil, err
	}

	if err := lifecycle.EnsureServerTransition(server.Status, "connected"); err != nil {
		return nil, err
	}
	prev := server.Status
	server.Status = "connected"
	if err := s.repo.Update(ctx, server); err != nil {
		return nil, err
	}
	s.writeAudit("server", server.ID, prev, server.Status, "ssh_connect_success", "connect")

	return s.repo.GetByID(ctx, server.ID)
}

// GetStats fetches live resource usage from the remote server via SSH.
// The server must be in a connected/ready state.
func (s *Service) GetStats(ctx context.Context, id string) (*domain.ServerStats, error) {
	server, err := s.repo.GetByID(ctx, strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}

	// Bound the SSH round-trip so a slow server never hangs the UI.
	sshCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Each sub-command is separated by "---" and uses "|| true" so a
	// single failure (e.g. missing binary) doesn't abort the whole script.
	// This prevents the "all zeros" symptom caused by an && chain breakage.
	script := strings.Join([]string{
		`nproc 2>/dev/null || echo 0`,
		`awk '{print $1}' /proc/loadavg 2>/dev/null || echo 0`,
		`free -m 2>/dev/null | awk '/^Mem:/{print $2}' || echo 0`,
		`free -m 2>/dev/null | awk '/^Mem:/{print $3}' || echo 0`,
		`df -h / 2>/dev/null | tail -1 | awk '{print $2,$3,$4,$5}' || echo '? ? ? ?'`,
		`uptime -p 2>/dev/null || uptime 2>/dev/null | awk -F'up ' '{print $2}' | awk -F',' '{print $1}' || echo unknown`,
	}, "; echo '---'; ")

	out, err := s.sshExecutor.RunOutput(sshCtx, *server, script)
	if err != nil {
		return nil, fmt.Errorf("fetch server stats: %w", err)
	}

	return parseStats(out), nil
}

func parseStats(raw string) *domain.ServerStats {
	lines := strings.Split(raw, "---")
	// flatten and split by newline
	var parts []string
	for _, l := range lines {
		for _, item := range strings.Split(strings.TrimSpace(l), "\n") {
			item = strings.TrimSpace(item)
			if item != "" {
				parts = append(parts, item)
			}
		}
	}

	stats := &domain.ServerStats{}

	if len(parts) > 0 {
		stats.CPUCores, _ = strconv.Atoi(parts[0])
	}
	if len(parts) > 1 {
		stats.LoadAvg1, _ = strconv.ParseFloat(parts[1], 64)
	}
	if len(parts) > 2 {
		stats.RAMTotalMB, _ = strconv.ParseInt(parts[2], 10, 64)
	}
	if len(parts) > 3 {
		stats.RAMUsedMB, _ = strconv.ParseInt(parts[3], 10, 64)
	}
	if len(parts) > 4 {
		disk := strings.Fields(parts[4])
		if len(disk) >= 4 {
			stats.DiskTotal = disk[0]
			stats.DiskUsed = disk[1]
			stats.DiskFree = disk[2]
			stats.DiskPct = disk[3]
		}
	}
	if len(parts) > 5 {
		stats.Uptime = parts[5]
	}

	return stats
}

func (s *Service) writeAudit(resourceType, resourceID, from, to, reason, taskID string) {
	if s.audit == nil || from == to {
		return
	}
	_ = s.audit.WriteStatusEvent(auditdomain.StatusEvent{
		ResourceType: resourceType,
		ResourceID:   resourceID,
		FromStatus:   from,
		ToStatus:     to,
		Reason:       reason,
		TaskID:       taskID,
	})
}
