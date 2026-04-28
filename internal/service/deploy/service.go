package deploy

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hibiken/asynq"

	auditdomain "github.com/your-org/ventopanel/internal/domain/audit"
	domain "github.com/your-org/ventopanel/internal/domain/deploy"
	"github.com/your-org/ventopanel/internal/domain/lifecycle"
	"github.com/your-org/ventopanel/internal/domain/tasklog"
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
	taskLogs   tasklog.Repository
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
	taskLogs tasklog.Repository,
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
		taskLogs:   taskLogs,
	}
}

func (s *Service) EnqueueDeploy(ctx context.Context, siteID string) error {
	payload, err := json.Marshal(DeploySitePayload{SiteID: strings.TrimSpace(siteID)})
	if err != nil {
		return err
	}

	// No TaskID: the distributed lock inside ExecuteDeploy prevents parallel runs.
	// Using TaskID caused "task ID conflicts" errors when a previous retry was still queued.
	_, err = s.client.EnqueueContext(ctx, asynq.NewTask(TaskDeploySite, payload))
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

	// Create task log entry.
	logEntry := &tasklog.TaskLog{SiteID: siteID, TaskType: "deploy"}
	if s.taskLogs != nil {
		_ = s.taskLogs.Create(ctx, logEntry)
	}
	var outputBuf strings.Builder
	finishLog := func(status, extra string) {
		if s.taskLogs == nil || logEntry.ID == "" {
			return
		}
		_ = s.taskLogs.Finish(context.Background(), logEntry.ID, status, outputBuf.String()+extra)
	}
	appendOutput := func(cmd, out string, err error) {
		outputBuf.WriteString("$ " + cmd + "\n")
		if out != "" {
			outputBuf.WriteString(out + "\n")
		}
		if err != nil {
			outputBuf.WriteString("ERROR: " + err.Error() + "\n")
		}
	}

	site, err := s.siteRepo.GetByID(ctx, siteID)
	if err != nil {
		finishLog("failed", "ERROR: "+err.Error())
		return err
	}

	server, err := s.serverRepo.GetByID(ctx, site.ServerID)
	if err != nil {
		finishLog("failed", "ERROR: "+err.Error())
		return err
	}

	if server.Status != "ready_for_deploy" && server.Status != "deployed" {
		msg := fmt.Sprintf("server %s is not ready (status=%s): run Provision first", server.ID, server.Status)
		finishLog("failed", "ERROR: "+msg)
		return fmt.Errorf(msg)
	}

	if err := lifecycle.EnsureSiteTransition(site.Status, "deploying"); err != nil {
		finishLog("failed", "ERROR: "+err.Error())
		return err
	}
	prev := site.Status
	site.Status = "deploying"
	if err := s.siteRepo.Update(ctx, site); err != nil {
		finishLog("failed", "ERROR: "+err.Error())
		return err
	}
	s.writeAudit("site", site.ID, prev, site.Status, "deploy_started", TaskDeploySite)

	if err := s.firewall.EnsureDefaultRules(ctx, server.Host); err != nil {
		appendOutput("firewall:ensure_default_rules", "", err)
		if transitionErr := lifecycle.EnsureSiteTransition(site.Status, "deploy_failed"); transitionErr != nil {
			finishLog("failed", "")
			return transitionErr
		}
		prevFailed := site.Status
		site.Status = "deploy_failed"
		if updateErr := s.siteRepo.Update(ctx, site); updateErr != nil {
			finishLog("failed", "")
			return updateErr
		}
		s.writeAudit("site", site.ID, prevFailed, site.Status, "deploy_failed_firewall", TaskDeploySite)
		finishLog("failed", "")
		return err
	}

	appPort := derivePort(site.ID)
	baseDir := fmt.Sprintf("/opt/ventopanel/sites/%s", site.ID)
	composeContent := composeTemplate(site, appPort)
	nginxContent := nginxTemplate(site.Domain, appPort)

	// Use base64-encoded writes to avoid heredoc issues over SSH exec channels.
	composeB64 := base64.StdEncoding.EncodeToString([]byte(composeContent))
	nginxB64 := base64.StdEncoding.EncodeToString([]byte(nginxContent))

	commands := []struct{ name, cmd string }{
		{"mkdir", fmt.Sprintf("mkdir -p %s", baseDir)},
		{"write_compose", fmt.Sprintf("echo %s | base64 -d > %s/docker-compose.yml", composeB64, baseDir)},
		{"docker_up", fmt.Sprintf("docker compose -f %s/docker-compose.yml up -d --pull missing 2>&1", baseDir)},
		{"write_nginx", fmt.Sprintf("echo %s | base64 -d > /etc/nginx/sites-available/vento_%s.conf", nginxB64, site.ID)},
		{"link_nginx", fmt.Sprintf("ln -sfn /etc/nginx/sites-available/vento_%s.conf /etc/nginx/sites-enabled/vento_%s.conf", site.ID, site.ID)},
		{"nginx_test", "nginx -t 2>&1"},
		{"nginx_reload", "systemctl reload nginx"},
	}

	var deployErr error
	for _, step := range commands {
		out, err := s.ssh.RunOutput(ctx, *server, step.cmd)
		appendOutput(step.name, out, err)
		if err != nil {
			deployErr = err
			break
		}
	}

	if deployErr != nil {
		if transitionErr := lifecycle.EnsureSiteTransition(site.Status, "deploy_failed"); transitionErr != nil {
			finishLog("failed", "")
			return transitionErr
		}
		prevFailed := site.Status
		site.Status = "deploy_failed"
		if updateErr := s.siteRepo.Update(ctx, site); updateErr != nil {
			finishLog("failed", "")
			return updateErr
		}
		s.writeAudit("site", site.ID, prevFailed, site.Status, "deploy_failed_runtime", TaskDeploySite)
		finishLog("failed", "")
		return deployErr
	}

	if err := s.ssl.IssueCertificate(ctx, *server, site.Domain); err != nil {
		appendOutput("ssl_issue", "", err)
		if transitionErr := lifecycle.EnsureSiteTransition(site.Status, "ssl_pending"); transitionErr != nil {
			finishLog("failed", "")
			return transitionErr
		}
		prevSSL := site.Status
		site.Status = "ssl_pending"
		if updateErr := s.siteRepo.Update(ctx, site); updateErr != nil {
			finishLog("failed", "")
			return updateErr
		}
		s.writeAudit("site", site.ID, prevSSL, site.Status, "deploy_ssl_pending", TaskDeploySite)
		if s.sslQueue != nil {
			_ = s.sslQueue.EnqueueIssue(ctx, site.ID)
		}
		finishLog("success", "")
		return nil
	}

	appendOutput("ssl_issue", "certificate issued", nil)

	if err := lifecycle.EnsureSiteTransition(site.Status, "deployed"); err != nil {
		finishLog("failed", "ERROR: "+err.Error())
		return err
	}
	prevDeployed := site.Status
	site.Status = "deployed"
	if err := s.siteRepo.Update(ctx, site); err != nil {
		finishLog("failed", "ERROR: "+err.Error())
		return err
	}
	s.writeAudit("site", site.ID, prevDeployed, site.Status, "deploy_success", TaskDeploySite)
	finishLog("success", "")
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

	// Use YAML block literal (|-) for command to avoid YAML plain-scalar
	// restrictions (": " in content, backslash-quote confusion, etc.).
	if strings.Contains(runtime, "php") {
		phpCmd := `printf '<?php echo "VentoPanel site"; ?>' > /var/www/html/index.php && php -S 0.0.0.0:8080 -t /var/www/html`
		return fmt.Sprintf("services:\n  app:\n    image: php:8.3-fpm-alpine\n    container_name: ventopanel_%s\n    restart: unless-stopped\n    command:\n      - sh\n      - -c\n      - %q\n    ports:\n      - \"%d:8080\"\n",
			site.ID, phpCmd, appPort)
	}

	nodeCmd := `printf 'const h=require("http");h.createServer((q,r)=>r.end("VentoPanel site")).listen(8080);' > /app/server.js && node /app/server.js`
	return fmt.Sprintf("services:\n  app:\n    image: node:20-alpine\n    container_name: ventopanel_%s\n    restart: unless-stopped\n    command:\n      - sh\n      - -c\n      - %q\n    ports:\n      - \"%d:8080\"\n",
		site.ID, nodeCmd, appPort)
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
