package provision

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/hibiken/asynq"

	auditdomain "github.com/your-org/ventopanel/internal/domain/audit"
	"github.com/your-org/ventopanel/internal/domain/lifecycle"
	domain "github.com/your-org/ventopanel/internal/domain/server"
	"github.com/your-org/ventopanel/internal/infra/lock"
)

const TaskProvisionServer = "server:provision"

type Service struct {
	repo   domain.Repository
	ssh    domain.SSHExecutor
	client *asynq.Client
	lock   lockManager
	audit  auditdomain.StatusEventWriter
}

type lockManager interface {
	Acquire(ctx context.Context, key string, ttl time.Duration) error
	Release(ctx context.Context, key string) error
}

type ProvisionServerPayload struct {
	ServerID string `json:"server_id"`
}

func NewService(repo domain.Repository, ssh domain.SSHExecutor, client *asynq.Client, lock lockManager, audit auditdomain.StatusEventWriter) *Service {
	return &Service{repo: repo, ssh: ssh, client: client, lock: lock, audit: audit}
}

func (s *Service) EnqueueProvision(ctx context.Context, serverID string) error {
	payload, err := json.Marshal(ProvisionServerPayload{ServerID: strings.TrimSpace(serverID)})
	if err != nil {
		return err
	}

	_, err = s.client.EnqueueContext(
		ctx,
		asynq.NewTask(TaskProvisionServer, payload),
		asynq.TaskID("provision:"+strings.TrimSpace(serverID)),
	)
	return err
}

func (s *Service) ExecuteProvision(ctx context.Context, payload ProvisionServerPayload) error {
	serverID := strings.TrimSpace(payload.ServerID)
	lockKey := "lock:server:provision:" + serverID
	if s.lock != nil {
		if err := s.lock.Acquire(ctx, lockKey, 15*time.Minute); err != nil {
			if errors.Is(err, lock.ErrLockAlreadyHeld) {
				return nil
			}
			return err
		}
		defer func() { _ = s.lock.Release(context.Background(), lockKey) }()
	}

	server, err := s.repo.GetByID(ctx, serverID)
	if err != nil {
		return err
	}

	if err := lifecycle.EnsureServerTransition(server.Status, "provisioning"); err != nil {
		return err
	}
	prev := server.Status
	server.Status = "provisioning"
	if err := s.repo.Update(ctx, server); err != nil {
		return err
	}
	s.writeAudit("server", server.ID, prev, server.Status, "provision_started", TaskProvisionServer)

	commands := []string{
		"export DEBIAN_FRONTEND=noninteractive",
		"apt-get update -y",
		"apt-get install -y ca-certificates curl gnupg lsb-release ufw nginx certbot python3-certbot-nginx",
		"install -m 0755 -d /etc/apt/keyrings",
		"sh -c \"curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg\"",
		"chmod a+r /etc/apt/keyrings/docker.gpg",
		"sh -c \"echo \\\"deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo $VERSION_CODENAME) stable\\\" > /etc/apt/sources.list.d/docker.list\"",
		"apt-get update -y",
		"apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin",
		"systemctl enable --now docker",
		"ufw allow OpenSSH",
		"ufw allow 'Nginx Full'",
		"ufw --force enable",
		"systemctl enable --now nginx",
	}

	if err := s.ssh.RunScript(ctx, *server, commands); err != nil {
		if transitionErr := lifecycle.EnsureServerTransition(server.Status, "provision_failed"); transitionErr != nil {
			return transitionErr
		}
		prevFailed := server.Status
		server.Status = "provision_failed"
		if updateErr := s.repo.Update(ctx, server); updateErr != nil {
			return updateErr
		}
		s.writeAudit("server", server.ID, prevFailed, server.Status, "provision_failed", TaskProvisionServer)

		return err
	}

	if err := lifecycle.EnsureServerTransition(server.Status, "ready_for_deploy"); err != nil {
		return err
	}
	prevReady := server.Status
	server.Status = "ready_for_deploy"
	if err := s.repo.Update(ctx, server); err != nil {
		return err
	}
	s.writeAudit("server", server.ID, prevReady, server.Status, "provision_ready", TaskProvisionServer)
	return nil
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
