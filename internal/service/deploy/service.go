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

	webRoot := fmt.Sprintf("/var/www/vento_%s", site.ID)
	htmlContent := staticHTML(site.Domain)
	nginxContent := nginxStaticTemplate(site.Domain, webRoot)

	// Use base64-encoded writes to avoid heredoc issues over SSH exec channels.
	htmlB64 := base64.StdEncoding.EncodeToString([]byte(htmlContent))
	nginxB64 := base64.StdEncoding.EncodeToString([]byte(nginxContent))

	commands := []struct{ name, cmd string }{
		{"mkdir", fmt.Sprintf("mkdir -p %s", webRoot)},
		{"write_html", fmt.Sprintf("echo %s | base64 -d > %s/index.html", htmlB64, webRoot)},
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


// staticHTML returns a minimal HTML landing page for the deployed site.
func staticHTML(domain string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head><meta charset="UTF-8"><title>%s</title>
<style>body{font-family:sans-serif;display:flex;align-items:center;justify-content:center;height:100vh;margin:0;background:#f8fafc}
.box{text-align:center;padding:2rem 3rem;border-radius:12px;background:#fff;box-shadow:0 2px 16px rgba(0,0,0,.08)}
h1{margin:0 0 .5rem;font-size:1.8rem}p{color:#64748b;margin:0}</style></head>
<body><div class="box"><h1>%s</h1><p>Deployed via VentoPanel</p></div></body>
</html>
`, domain, domain)
}

// nginxStaticTemplate serves files from webRoot directly — no Docker, no rate limits.
func nginxStaticTemplate(domain, webRoot string) string {
	return fmt.Sprintf(`server {
    listen 80;
    server_name %s;

    root %s;
    index index.html;

    location / {
        try_files $uri $uri/ =404;
    }
}
`, domain, webRoot)
}
