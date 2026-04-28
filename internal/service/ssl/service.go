package ssl

import (
	"context"
	"encoding/json"
	"errors"
	"math/rand"
	"sync"
	"strings"
	"time"

	"github.com/hibiken/asynq"

	auditdomain "github.com/your-org/ventopanel/internal/domain/audit"
	deploydomain "github.com/your-org/ventopanel/internal/domain/deploy"
	"github.com/your-org/ventopanel/internal/domain/lifecycle"
	"github.com/your-org/ventopanel/internal/infra/lock"
	"github.com/your-org/ventopanel/internal/infra/metrics"
)

const TaskIssueSSL = "ssl:issue"
const TaskRenewSSL = "ssl:renew"

var ErrQueueUnavailable = errors.New("ssl queue client is not configured")

// SSLCertInfo holds certificate expiry information for a site.
type SSLCertInfo struct {
	Domain    string    `json:"domain"`
	ExpiresAt time.Time `json:"expires_at"`
	DaysLeft  int       `json:"days_left"`
	Status    string    `json:"status"` // valid | expiring_soon | expired | no_cert
}

type Service struct {
	siteRepo   deploydomain.SiteRepository
	serverRepo deploydomain.ServerRepository
	ssl        deploydomain.SSLManager
	ssh        deploydomain.SSHExecutor
	client     *asynq.Client
	lock       lockManager
	audit      auditdomain.StatusEventWriter
	mu         sync.RWMutex
	stats      Stats
}

type lockManager interface {
	Acquire(ctx context.Context, key string, ttl time.Duration) error
	Release(ctx context.Context, key string) error
}

type Stats struct {
	ScheduledRenewTotal   uint64     `json:"scheduled_renew_total"`
	SuccessfulRenewTotal  uint64     `json:"successful_renew_total"`
	FailedRenewTotal      uint64     `json:"failed_renew_total"`
	LastBatchEnqueuedAt   *time.Time `json:"last_batch_enqueued_at,omitempty"`
	LastBatchServerCount  int        `json:"last_batch_server_count"`
	LastBatchError        string     `json:"last_batch_error,omitempty"`
}

type IssueSSLPayload struct {
	SiteID string `json:"site_id"`
}

type RenewSSLPayload struct {
	ServerID string `json:"server_id"`
}

func NewService(
	siteRepo deploydomain.SiteRepository,
	serverRepo deploydomain.ServerRepository,
	sslManager deploydomain.SSLManager,
	client *asynq.Client,
	lock lockManager,
	audit auditdomain.StatusEventWriter,
) *Service {
	metrics.Register()

	return &Service{
		siteRepo:   siteRepo,
		serverRepo: serverRepo,
		ssl:        sslManager,
		client:     client,
		lock:       lock,
		audit:      audit,
	}
}

// WithSSH enables certificate info lookups via SSH.
func (s *Service) WithSSH(executor deploydomain.SSHExecutor) *Service {
	s.ssh = executor
	return s
}

// GetCertInfo fetches SSL certificate expiry for the given site via SSH.
func (s *Service) GetCertInfo(ctx context.Context, siteID string) (*SSLCertInfo, error) {
	site, err := s.siteRepo.GetByID(ctx, siteID)
	if err != nil {
		return nil, err
	}

	server, err := s.serverRepo.GetByID(ctx, site.ServerID)
	if err != nil {
		return nil, err
	}

	info := &SSLCertInfo{Domain: site.Domain, Status: "no_cert"}

	if s.ssh == nil {
		return info, nil
	}

	cmd := "openssl x509 -enddate -noout -in /etc/letsencrypt/live/" + site.Domain + "/fullchain.pem 2>/dev/null || echo no_cert"
	out, err := s.ssh.RunOutput(ctx, *server, cmd)
	if err != nil || strings.TrimSpace(out) == "no_cert" || out == "" {
		return info, nil
	}

	// Output format: "notAfter=May 28 12:00:00 2026 GMT"
	out = strings.TrimSpace(out)
	out = strings.TrimPrefix(out, "notAfter=")
	expiry, err := time.Parse("Jan _2 15:04:05 2006 MST", out)
	if err != nil {
		expiry, err = time.Parse("Jan  2 15:04:05 2006 MST", out)
	}
	if err != nil {
		return info, nil
	}

	info.ExpiresAt = expiry
	info.DaysLeft = int(time.Until(expiry).Hours() / 24)
	switch {
	case info.DaysLeft < 0:
		info.Status = "expired"
	case info.DaysLeft <= 30:
		info.Status = "expiring_soon"
	default:
		info.Status = "valid"
	}

	return info, nil
}

func (s *Service) EnqueueIssue(ctx context.Context, siteID string) error {
	if s.client == nil {
		return ErrQueueUnavailable
	}

	payload, err := json.Marshal(IssueSSLPayload{SiteID: strings.TrimSpace(siteID)})
	if err != nil {
		return err
	}

	_, err = s.client.EnqueueContext(
		ctx,
		asynq.NewTask(TaskIssueSSL, payload),
		asynq.TaskID("ssl-issue:"+strings.TrimSpace(siteID)),
		asynq.MaxRetry(7),
		asynq.ProcessIn(30*time.Second),
	)
	return err
}

func (s *Service) ExecuteIssue(ctx context.Context, payload IssueSSLPayload) error {
	siteID := strings.TrimSpace(payload.SiteID)
	lockKey := "lock:site:ssl-issue:" + siteID
	if s.lock != nil {
		if err := s.lock.Acquire(ctx, lockKey, 10*time.Minute); err != nil {
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

	if err := s.ssl.IssueCertificate(ctx, *server, site.Domain); err != nil {
		if transitionErr := lifecycle.EnsureSiteTransition(site.Status, "ssl_pending"); transitionErr != nil {
			return transitionErr
		}
		site.Status = "ssl_pending"
		if updateErr := s.siteRepo.Update(ctx, site); updateErr != nil {
			return updateErr
		}
		s.writeAudit("site", site.ID, "deployed", site.Status, "ssl_issue_failed", TaskIssueSSL)
		return err
	}

	if err := lifecycle.EnsureSiteTransition(site.Status, "deployed"); err != nil {
		return err
	}
	site.Status = "deployed"
	if err := s.siteRepo.Update(ctx, site); err != nil {
		return err
	}
	s.writeAudit("site", site.ID, "ssl_pending", site.Status, "ssl_issue_success", TaskIssueSSL)
	return nil
}

func (s *Service) EnqueueRenew(ctx context.Context, serverID string) error {
	return s.enqueueRenew(ctx, strings.TrimSpace(serverID), 10*time.Second)
}

func (s *Service) EnqueueDailyRenewForAll(ctx context.Context, maxJitter time.Duration) error {
	servers, err := s.serverRepo.List(ctx)
	if err != nil {
		s.setBatchError(err.Error())
		return err
	}

	jitter := maxJitter
	if jitter < 0 {
		jitter = 0
	}

	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	for _, server := range servers {
		delay := 0 * time.Second
		if jitter > 0 {
			delay = time.Duration(random.Int63n(int64(jitter)))
		}

		if err := s.enqueueRenew(ctx, server.ID, delay); err != nil {
			s.setBatchError(err.Error())
			return err
		}
	}

	now := time.Now().UTC()
	s.mu.Lock()
	s.stats.LastBatchEnqueuedAt = &now
	s.stats.LastBatchServerCount = len(servers)
	s.stats.LastBatchError = ""
	s.mu.Unlock()
	metrics.SetLastBatchServerCount(len(servers))

	return nil
}

func (s *Service) enqueueRenew(ctx context.Context, serverID string, delay time.Duration) error {
	if s.client == nil {
		return ErrQueueUnavailable
	}

	payload, err := json.Marshal(RenewSSLPayload{ServerID: strings.TrimSpace(serverID)})
	if err != nil {
		return err
	}

	_, err = s.client.EnqueueContext(
		ctx,
		asynq.NewTask(TaskRenewSSL, payload),
		asynq.TaskID("ssl-renew:"+strings.TrimSpace(serverID)),
		asynq.MaxRetry(5),
		asynq.ProcessIn(delay),
	)
	if err == nil {
		s.mu.Lock()
		s.stats.ScheduledRenewTotal++
		s.mu.Unlock()
		metrics.IncSSLRenewScheduled()
	}
	return err
}

func (s *Service) ExecuteRenew(ctx context.Context, payload RenewSSLPayload) error {
	serverID := strings.TrimSpace(payload.ServerID)
	lockKey := "lock:server:ssl-renew:" + serverID
	if s.lock != nil {
		if err := s.lock.Acquire(ctx, lockKey, 10*time.Minute); err != nil {
			if errors.Is(err, lock.ErrLockAlreadyHeld) {
				return nil
			}
			s.incRenewFailed()
			return err
		}
		defer func() { _ = s.lock.Release(context.Background(), lockKey) }()
	}

	server, err := s.serverRepo.GetByID(ctx, serverID)
	if err != nil {
		s.incRenewFailed()
		return err
	}

	if err := s.ssl.RenewCertificates(ctx, *server); err != nil {
		now := time.Now().UTC()
		server.LastRenewAt = &now
		server.LastRenewStatus = "failed"
		if updateErr := s.serverRepo.Update(ctx, server); updateErr != nil {
			s.incRenewFailed()
			return updateErr
		}
		s.writeAudit("server", server.ID, "renewing", "renew_failed", "ssl_renew_failed", TaskRenewSSL)
		s.incRenewFailed()
		return err
	}

	now := time.Now().UTC()
	server.LastRenewAt = &now
	server.LastRenewStatus = "success"
	if err := s.serverRepo.Update(ctx, server); err != nil {
		s.incRenewFailed()
		return err
	}
	s.writeAudit("server", server.ID, "renewing", "renew_success", "ssl_renew_success", TaskRenewSSL)

	s.mu.Lock()
	s.stats.SuccessfulRenewTotal++
	s.mu.Unlock()
	metrics.IncSSLRenewSuccess()

	return nil
}

func (s *Service) Stats() Stats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.stats
}

func (s *Service) setBatchError(msg string) {
	s.mu.Lock()
	s.stats.LastBatchError = msg
	s.mu.Unlock()
}

func (s *Service) incRenewFailed() {
	s.mu.Lock()
	s.stats.FailedRenewTotal++
	s.mu.Unlock()
	metrics.IncSSLRenewFailed()
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
