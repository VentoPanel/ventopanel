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
	pgrepo "github.com/your-org/ventopanel/internal/repository/postgres"
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
	envRepo    *pgrepo.EnvRepository
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
	envRepo *pgrepo.EnvRepository,
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
		envRepo:    envRepo,
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
		// Wrap with SkipRetry so Asynq stops retrying — the site is already
		// in a state that cannot transition to deploying (e.g. ssl_pending).
		return fmt.Errorf("%w: %w", asynq.SkipRetry, err)
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

	var commands []struct{ name, cmd string }

	if strings.TrimSpace(site.RepositoryURL) != "" {
		// Git-based deploy: clone/pull → detect runtime → docker build → run.
		appDir := fmt.Sprintf("/opt/ventopanel/sites/%s/app", site.ID)
		appPort := derivePort(site.ID)
		envFlags := s.buildEnvFlags(ctx, site.ID)
		branch := site.Branch
		if branch == "" {
			branch = "main"
		}
		script := repoDeployScript(site.ID, site.RepositoryURL, appDir, branch, appPort, envFlags)
		scriptB64 := base64.StdEncoding.EncodeToString([]byte(script))
		nginxContent := nginxProxyTemplate(site.Domain, appPort)
		nginxB64 := base64.StdEncoding.EncodeToString([]byte(nginxContent))

		commands = []struct{ name, cmd string }{
			{"deploy_app", fmt.Sprintf("echo %s | base64 -d | sh 2>&1", scriptB64)},
			{"write_nginx", fmt.Sprintf("echo %s | base64 -d > /etc/nginx/sites-available/vento_%s.conf", nginxB64, site.ID)},
			{"link_nginx", fmt.Sprintf("ln -sfn /etc/nginx/sites-available/vento_%s.conf /etc/nginx/sites-enabled/vento_%s.conf", site.ID, site.ID)},
			{"nginx_test", "nginx -t 2>&1"},
			{"nginx_reload", "systemctl reload nginx"},
		}
	} else {
		// Static placeholder deploy: write HTML + nginx root config.
		webRoot := fmt.Sprintf("/var/www/vento_%s", site.ID)
		htmlContent := staticHTML(site.Domain)
		nginxContent := nginxStaticTemplate(site.Domain, webRoot)
		htmlB64 := base64.StdEncoding.EncodeToString([]byte(htmlContent))
		nginxB64 := base64.StdEncoding.EncodeToString([]byte(nginxContent))

		commands = []struct{ name, cmd string }{
			{"mkdir", fmt.Sprintf("mkdir -p %s", webRoot)},
			{"write_html", fmt.Sprintf("echo %s | base64 -d > %s/index.html", htmlB64, webRoot)},
			{"write_nginx", fmt.Sprintf("echo %s | base64 -d > /etc/nginx/sites-available/vento_%s.conf", nginxB64, site.ID)},
			{"link_nginx", fmt.Sprintf("ln -sfn /etc/nginx/sites-available/vento_%s.conf /etc/nginx/sites-enabled/vento_%s.conf", site.ID, site.ID)},
			{"nginx_test", "nginx -t 2>&1"},
			{"nginx_reload", "systemctl reload nginx"},
		}
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

// ServerContainer is one row from `docker ps` on the remote server.
type ServerContainer struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Ports  string `json:"ports"`
	Image  string `json:"image"`
}

// GetServerContainers runs `docker ps` on a server and returns all
// ventopanel-managed containers (name prefix ventopanel_).
func (s *Service) GetServerContainers(ctx context.Context, serverID string) ([]ServerContainer, error) {
	server, err := s.serverRepo.GetByID(ctx, serverID)
	if err != nil {
		return nil, err
	}

	sshCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Format: name|status|ports|image, one per line, only ventopanel_ containers.
	cmd := `docker ps -a --filter 'name=ventopanel_' --format '{{.Names}}|{{.Status}}|{{.Ports}}|{{.Image}}'`
	out, sshErr := s.ssh.RunOutput(sshCtx, *server, cmd)
	if sshErr != nil {
		return nil, fmt.Errorf("docker ps: %w", sshErr)
	}
	out = strings.TrimSpace(out)
	if out == "" {
		return []ServerContainer{}, nil
	}
	lines := strings.Split(out, "\n")
	result := make([]ServerContainer, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 4)
		for len(parts) < 4 {
			parts = append(parts, "")
		}
		result = append(result, ServerContainer{
			Name:   parts[0],
			Status: parts[1],
			Ports:  parts[2],
			Image:  parts[3],
		})
	}
	return result, nil
}

// ContainerInfo holds live runtime data for a site's Docker container.
type ContainerInfo struct {
	Status    string `json:"status"`      // running | exited | not_found | no_container
	StartedAt string `json:"started_at"`
	CPUPerc   string `json:"cpu_percent"`
	MemUsage  string `json:"mem_usage"`
}

// GetContainerInfo returns live Docker container stats for a git-deployed site.
// Uses two separate SSH calls so each command is simple and debuggable.
func (s *Service) GetContainerInfo(ctx context.Context, siteID string) (*ContainerInfo, error) {
	site, err := s.siteRepo.GetByID(ctx, siteID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(site.RepositoryURL) == "" {
		return &ContainerInfo{Status: "no_container"}, nil
	}
	server, err := s.serverRepo.GetByID(ctx, site.ServerID)
	if err != nil {
		return nil, err
	}

	name := fmt.Sprintf("ventopanel_%s", siteID)

	// 1. docker inspect — instant, works for any state (running/exited/stopped).
	inspectOut, _ := s.ssh.RunOutput(ctx, *server,
		fmt.Sprintf(`docker inspect --format '{{.State.Status}}|{{.State.StartedAt}}' %s 2>/dev/null || echo 'not_found|'`, name))
	inspectOut = strings.TrimSpace(inspectOut)

	parts := strings.SplitN(inspectOut, "|", 2)
	for len(parts) < 2 {
		parts = append(parts, "")
	}
	info := &ContainerInfo{
		Status:    strings.TrimSpace(parts[0]),
		StartedAt: strings.TrimSpace(parts[1]),
	}
	if info.Status == "" {
		info.Status = "not_found"
	}

	// 2. docker stats — only meaningful when the container is running.
	// `docker stats --no-stream` computes CPU% as a delta between two kernel
	// readings. The daemon needs at least two measurement points separated in
	// time. We take two back-to-back readings with a 1-second gap; the first
	// primes the daemon cache, the second returns the actual current value.
	if info.Status == "running" {
		statsCmd := fmt.Sprintf(
			`docker stats --no-stream --format '{{.CPUPerc}}|{{.MemUsage}}' %s 2>/dev/null; `+
				`sleep 1; `+
				`docker stats --no-stream --format '{{.CPUPerc}}|{{.MemUsage}}' %s 2>/dev/null || echo '|'`,
			name, name,
		)
		statsOut, _ := s.ssh.RunOutput(ctx, *server, statsCmd)
		// Take the last non-empty line (second reading).
		var lastLine string
		for _, line := range strings.Split(strings.TrimSpace(statsOut), "\n") {
			if l := strings.TrimSpace(line); l != "" {
				lastLine = l
			}
		}
		sp := strings.SplitN(lastLine, "|", 2)
		for len(sp) < 2 {
			sp = append(sp, "")
		}
		info.CPUPerc = strings.TrimSpace(sp[0])
		info.MemUsage = strings.TrimSpace(sp[1])
	}

	return info, nil
}

// GetContainerLogs returns the last n lines of Docker container logs.
func (s *Service) GetContainerLogs(ctx context.Context, siteID string, tail int) (string, error) {
	site, err := s.siteRepo.GetByID(ctx, siteID)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(site.RepositoryURL) == "" {
		return "(static site — no container)", nil
	}
	server, err := s.serverRepo.GetByID(ctx, site.ServerID)
	if err != nil {
		return "", err
	}
	if tail <= 0 {
		tail = 100
	}
	cmd := fmt.Sprintf("docker logs --tail %d ventopanel_%s 2>&1", tail, siteID)
	out, _ := s.ssh.RunOutput(ctx, *server, cmd)
	return out, nil
}

// RestartContainer stops the old container and starts a fresh one with
// the current env vars from the database (without rebuilding the image).
func (s *Service) RestartContainer(ctx context.Context, siteID string) error {
	site, err := s.siteRepo.GetByID(ctx, siteID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(site.RepositoryURL) == "" {
		return fmt.Errorf("site has no container (static deploy)")
	}
	server, err := s.serverRepo.GetByID(ctx, site.ServerID)
	if err != nil {
		return err
	}
	name := fmt.Sprintf("ventopanel_%s", siteID)
	appPort := derivePort(siteID)
	envFlags := s.buildEnvFlags(ctx, siteID)

	// Remove old container and start a fresh one so new env vars take effect.
	script := fmt.Sprintf(`docker rm -f %s 2>/dev/null || true
docker run -d --name %s --restart unless-stopped -p %d:3000%s %s`, name, name, appPort, envFlags, name)
	_, err = s.ssh.RunOutput(ctx, *server, script)
	return err
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


// buildEnvFlags loads site env vars from the DB and returns a shell-safe
// "-e KEY=VALUE" string for docker run. Values containing spaces or special
// characters are quoted with single quotes (single quotes inside are escaped).
func (s *Service) buildEnvFlags(ctx context.Context, siteID string) string {
	if s.envRepo == nil {
		return ""
	}
	vars, err := s.envRepo.ListBySiteID(ctx, siteID)
	if err != nil || len(vars) == 0 {
		return ""
	}
	var sb strings.Builder
	for _, v := range vars {
		escaped := strings.ReplaceAll(v.Value, "'", "'\\''")
		sb.WriteString(fmt.Sprintf(" -e %s='%s'", v.Key, escaped))
	}
	return sb.String()
}

func derivePort(siteID string) int {
	sum := 0
	for _, r := range siteID {
		sum += int(r)
	}
	return 20000 + (sum % 10000)
}

// repoDeployScript returns a POSIX shell script that:
//  1. Clones or updates the git repo
//  2. Auto-detects runtime and generates a Dockerfile when missing
//  3. Builds the image locally (no Docker Hub pull for the app image)
//  4. Replaces the running container
//
// The script is designed to be piped to sh via base64:
//
//	echo BASE64 | base64 -d | sh 2>&1
func repoDeployScript(siteID, repoURL, appDir, branch string, appPort int, envFlags string) string {
	return fmt.Sprintf(`#!/bin/sh
set -e

APP_DIR="%s"
SITE_ID="%s"
REPO_URL="%s"
BRANCH="%s"
PORT=%d

mkdir -p "$APP_DIR"

# Clone repo on first deploy; pull latest on re-deploy.
if [ -d "$APP_DIR/.git" ]; then
  CURRENT_REMOTE=$(git -C "$APP_DIR" remote get-url origin 2>/dev/null || echo "")
  if [ "$CURRENT_REMOTE" = "$REPO_URL" ]; then
    echo "==> Updating branch $BRANCH..."
    git -C "$APP_DIR" fetch --depth 1 origin "$BRANCH"
    git -C "$APP_DIR" reset --hard FETCH_HEAD
  else
    echo "==> Repo URL changed, re-cloning branch $BRANCH..."
    rm -rf "$APP_DIR"
    git clone --depth 1 --branch "$BRANCH" "$REPO_URL" "$APP_DIR"
  fi
else
  echo "==> Cloning branch $BRANCH..."
  git clone --depth 1 --branch "$BRANCH" "$REPO_URL" "$APP_DIR"
fi

cd "$APP_DIR"

# Auto-detect runtime and generate Dockerfile when none exists.
if [ -f Dockerfile ]; then
  echo "==> Using existing Dockerfile"
elif [ -f package.json ]; then
  echo "==> Detected Node.js — generating Dockerfile"
  printf 'FROM node:20-alpine\nWORKDIR /app\nCOPY package*.json ./\nRUN npm install --production 2>&1 || npm install 2>&1\nCOPY . .\nEXPOSE 3000\nCMD ["npm","start"]\n' > Dockerfile
elif [ -f requirements.txt ]; then
  echo "==> Detected Python — generating Dockerfile"
  printf 'FROM python:3.12-slim\nWORKDIR /app\nCOPY requirements.txt .\nRUN pip install --no-cache-dir -r requirements.txt\nCOPY . .\nEXPOSE 3000\nCMD ["python","app.py"]\n' > Dockerfile
elif [ -f go.mod ]; then
  echo "==> Detected Go — generating Dockerfile"
  printf 'FROM golang:1.22-alpine AS builder\nWORKDIR /app\nCOPY . .\nRUN go build -o server ./...\nFROM alpine:latest\nCOPY --from=builder /app/server /server\nEXPOSE 3000\nCMD ["/server"]\n' > Dockerfile
elif [ -f composer.json ]; then
  echo "==> Detected PHP — generating Dockerfile"
  printf 'FROM php:8.3-cli-alpine\nWORKDIR /app\nCOPY . .\nEXPOSE 3000\nCMD ["php","-S","0.0.0.0:3000","-t","."]\n' > Dockerfile
else
  echo "ERROR: No Dockerfile, package.json, requirements.txt, go.mod, or composer.json found."
  echo "Please add a Dockerfile to your repository."
  exit 1
fi

echo "==> Building Docker image..."
docker build -t "ventopanel_${SITE_ID}" .

echo "==> Stopping old container (if any)..."
docker rm -f "ventopanel_${SITE_ID}" 2>/dev/null || true

echo "==> Starting container on port ${PORT}..."
docker run -d \
  --name "ventopanel_${SITE_ID}" \
  --restart unless-stopped \
  -p "${PORT}:3000"%s \
  "ventopanel_${SITE_ID}"

echo "==> Done."
`, appDir, siteID, repoURL, branch, appPort, envFlags)
}

// nginxProxyTemplate proxies requests to a Docker container on appPort.
func nginxProxyTemplate(domain string, appPort int) string {
	return fmt.Sprintf(`server {
    listen 80;
    server_name %s;

    location / {
        proxy_pass http://127.0.0.1:%d;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 60s;
    }
}
`, domain, appPort)
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
