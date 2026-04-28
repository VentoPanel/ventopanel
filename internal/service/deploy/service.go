package deploy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hibiken/asynq"

	auditdomain "github.com/your-org/ventopanel/internal/domain/audit"
	domain "github.com/your-org/ventopanel/internal/domain/deploy"
	"github.com/your-org/ventopanel/internal/domain/lifecycle"
	"github.com/your-org/ventopanel/internal/infra/lock"
	sitedomain "github.com/your-org/ventopanel/internal/domain/site"
)

const TaskDeploySite = "deploy:site"

type Service struct {
	siteRepo   domain.SiteRepository
	serverRepo domain.ServerRepository
	ssh        domain.SSHExecutor
	firewall   domain.FirewallManager
	ssl        domain.SSLManager
	sslQueue   sslQueue
	client     *asynq.Client
	lock       lockManager
	audit      auditdomain.StatusEventWriter
}

type sslQueue interface {
	EnqueueIssue(ctx context.Context, siteID string) error
}

type lockManager interface {
	Acquire(ctx context.Context, key string, ttl time.Duration) error
	Release(ctx context.Context, key string) error
}

type DeploySitePayload struct {
	SiteID string `json:"site_id"`
}

func NewService(
	siteRepo domain.SiteRepository,
	serverRepo domain.ServerRepository,
	sshExecutor domain.SSHExecutor,
	firewallManager domain.FirewallManager,
	sslManager domain.SSLManager,
	sslQueue sslQueue,
	client *asynq.Client,
	lock lockManager,
	audit auditdomain.StatusEventWriter,
) *Service {
	return &Service{
		siteRepo:   siteRepo,
		serverRepo: serverRepo,
		ssh:        sshExecutor,
		firewall:   firewallManager,
		ssl:        sslManager,
		sslQueue:   sslQueue,
		client:     client,
		lock:       lock,
		audit:      audit,
	}
}

func (s *Service) EnqueueDeploy(ctx context.Context, siteID string) error {
	payload, err := json.Marshal(DeploySitePayload{SiteID: strings.TrimSpace(siteID)})
	if err != nil {
		return err
	}

	_, err = s.client.EnqueueContext(
		ctx,
		asynq.NewTask(TaskDeploySite, payload),
		asynq.TaskID("deploy:"+strings.TrimSpace(siteID)),
	)
	return err
}

func (s *Service) ExecuteDeploy(ctx context.Context, payload DeploySitePayload) error {
	siteID := strings.TrimSpace(payload.SiteID)
	lockKey := "lock:site:deploy:" + siteID
	if s.lock != nil {
		if err := s.lock.Acquire(ctx, lockKey, 15*time.Minute); err != nil {
			if errors.Is(err, lock.ErrLockAlreadyHeld) {
				return nil
			}
			return err
		}
		defer func() { _ = s.lock.Release(context.Background(), lockKey) }()
	}

	site, err := s.siteRepo.GetByID(ctx, siteID)
	if err != nil {
		return err
	}

	server, err := s.serverRepo.GetByID(ctx, site.ServerID)
	if err != nil {
		return err
	}

	if err := lifecycle.EnsureSiteTransition(site.Status, "deploying"); err != nil {
		return err
	}
	prev := site.Status
	site.Status = "deploying"
	if err := s.siteRepo.Update(ctx, site); err != nil {
		return err
	}
	s.writeAudit("site", site.ID, prev, site.Status, "deploy_started", TaskDeploySite)

	if err := s.firewall.EnsureDefaultRules(ctx, server.Host); err != nil {
		if transitionErr := lifecycle.EnsureSiteTransition(site.Status, "deploy_failed"); transitionErr != nil {
			return transitionErr
		}
		prevFailed := site.Status
		site.Status = "deploy_failed"
		if updateErr := s.siteRepo.Update(ctx, site); updateErr != nil {
			return updateErr
		}
		s.writeAudit("site", site.ID, prevFailed, site.Status, "deploy_failed_firewall", TaskDeploySite)
		return err
	}

	appPort := derivePort(site.ID)
	baseDir := fmt.Sprintf("/opt/ventopanel/sites/%s", site.ID)

	composeContent := composeTemplate(site, appPort)
	nginxContent := nginxTemplate(site.Domain, appPort)

	commands := []string{
		fmt.Sprintf("mkdir -p %s", baseDir),
		fmt.Sprintf("cat > %s/docker-compose.yml <<'EOF'\n%s\nEOF", baseDir, composeContent),
		fmt.Sprintf("docker compose -f %s/docker-compose.yml up -d", baseDir),
		fmt.Sprintf("cat > /etc/nginx/sites-available/vento_%s.conf <<'EOF'\n%s\nEOF", site.ID, nginxContent),
		fmt.Sprintf("ln -sfn /etc/nginx/sites-available/vento_%s.conf /etc/nginx/sites-enabled/vento_%s.conf", site.ID, site.ID),
		"nginx -t",
		"systemctl reload nginx",
	}

	if err := s.ssh.RunScript(ctx, *server, commands); err != nil {
		if transitionErr := lifecycle.EnsureSiteTransition(site.Status, "deploy_failed"); transitionErr != nil {
			return transitionErr
		}
		prevFailed := site.Status
		site.Status = "deploy_failed"
		if updateErr := s.siteRepo.Update(ctx, site); updateErr != nil {
			return updateErr
		}
		s.writeAudit("site", site.ID, prevFailed, site.Status, "deploy_failed_runtime", TaskDeploySite)
		return err
	}

	if err := s.ssl.IssueCertificate(ctx, *server, site.Domain); err != nil {
		if transitionErr := lifecycle.EnsureSiteTransition(site.Status, "ssl_pending"); transitionErr != nil {
			return transitionErr
		}
		prevSSL := site.Status
		site.Status = "ssl_pending"
		if updateErr := s.siteRepo.Update(ctx, site); updateErr != nil {
			return updateErr
		}
		s.writeAudit("site", site.ID, prevSSL, site.Status, "deploy_ssl_pending", TaskDeploySite)
		if s.sslQueue != nil {
			_ = s.sslQueue.EnqueueIssue(ctx, site.ID)
		}
		return nil
	}

	if err := lifecycle.EnsureSiteTransition(site.Status, "deployed"); err != nil {
		return err
	}
	prevDeployed := site.Status
	site.Status = "deployed"
	if err := s.siteRepo.Update(ctx, site); err != nil {
		return err
	}
	s.writeAudit("site", site.ID, prevDeployed, site.Status, "deploy_success", TaskDeploySite)
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

func derivePort(siteID string) int {
	sum := 0
	for _, r := range siteID {
		sum += int(r)
	}

	return 20000 + (sum % 10000)
}

func composeTemplate(site *sitedomain.Site, appPort int) string {
	runtime := strings.ToLower(strings.TrimSpace(site.Runtime))
	if strings.Contains(runtime, "php") {
		return fmt.Sprintf(`services:
  app:
    image: php:8.3-fpm-alpine
    container_name: ventopanel_%s
    restart: unless-stopped
    command: sh -c "printf '<?php echo \"VentoPanel PHP site: %s\"; ?>' > /var/www/html/index.php && php -S 0.0.0.0:8080 -t /var/www/html"
    ports:
      - "%d:8080"
`, site.ID, site.Domain, appPort)
	}

	return fmt.Sprintf(`services:
  app:
    image: node:20-alpine
    container_name: ventopanel_%s
    restart: unless-stopped
    command: sh -c "printf 'const http=require(\"http\");http.createServer((req,res)=>res.end(\"VentoPanel Node site: %s\")).listen(8080);' > /app/server.js && node /app/server.js"
    ports:
      - "%d:8080"
`, site.ID, site.Domain, appPort)
}

func nginxTemplate(domainName string, appPort int) string {
	return fmt.Sprintf(`server {
    listen 80;
    server_name %s;

    location / {
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_pass http://127.0.0.1:%d;
    }
}
`, domainName, appPort)
}
