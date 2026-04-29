package uptime

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	settingsdomain "github.com/your-org/ventopanel/internal/domain/settings"
	pgrepo "github.com/your-org/ventopanel/internal/repository/postgres"
)

type notifier interface {
	NotifyAll(ctx context.Context, message string) error
}

type settingsReader interface {
	GetNotificationConfig(ctx context.Context) (settingsdomain.NotificationConfig, error)
}

type siteNotifyState struct {
	failStreak int
	okStreak   int
	incident   bool // DOWN alert sent; waiting for recovery threshold
}

type Service struct {
	siteRepo    *pgrepo.SiteRepository
	uptimeRepo  *pgrepo.UptimeRepository
	notifier    notifier
	settings    settingsReader
	client      *http.Client
	mu          sync.Mutex
	notifyState map[string]*siteNotifyState
}

// concurrency limit: at most N parallel HTTP checks.
const maxConcurrent = 10

func NewService(
	siteRepo *pgrepo.SiteRepository,
	uptimeRepo *pgrepo.UptimeRepository,
	notifier notifier,
	settings settingsReader,
) *Service {
	return &Service{
		siteRepo:    siteRepo,
		uptimeRepo:  uptimeRepo,
		notifier:    notifier,
		settings:    settings,
		notifyState: make(map[string]*siteNotifyState),
		client: &http.Client{
			Timeout: 10 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 5 {
					return http.ErrUseLastResponse
				}
				return nil
			},
		},
	}
}

// CheckAll pings every active site concurrently (bounded by maxConcurrent).
// Config is fetched once per cycle so we don't hammer the DB N×.
func (s *Service) CheckAll(ctx context.Context) {
	sites, err := s.siteRepo.List(ctx)
	if err != nil {
		return
	}

	// Fetch notification config once for the whole cycle.
	cfg := settingsdomain.NotificationConfig{
		UptimeNotifyDown:        true,
		UptimeNotifyRecovery:    true,
		UptimeFailThreshold:     1,
		UptimeRecoveryThreshold: 1,
	}
	if s.settings != nil {
		if c, err := s.settings.GetNotificationConfig(ctx); err == nil {
			cfg = c
		}
	}

	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup

	for _, site := range sites {
		switch site.Status {
		case "deployed", "ssl_pending", "ssl_active":
		default:
			continue
		}
		if strings.TrimSpace(site.Domain) == "" {
			continue
		}

		hcPath := strings.TrimSpace(site.HealthcheckPath)
		if hcPath == "" {
			hcPath = "/"
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(id, domain, path string) {
			defer wg.Done()
			defer func() { <-sem }()
			s.checkSite(ctx, id, domain, path, cfg)
		}(site.ID, site.Domain, hcPath)
	}

	wg.Wait()
}

func (s *Service) checkSite(ctx context.Context, siteID, domain, hcPath string, cfg settingsdomain.NotificationConfig) {
	status, latency, statusCode, checkErr := s.ping(ctx, domain, hcPath)

	check := pgrepo.UptimeCheck{SiteID: siteID, Status: status, LatencyMs: latency, StatusCode: statusCode, Error: checkErr}

	firstEver, _ := s.uptimeRepo.IsFirstCheck(ctx, siteID)

	_ = s.uptimeRepo.Insert(ctx, check)

	failTh := settingsdomain.ClampInt(cfg.UptimeFailThreshold, 1, 60)
	recoverTh := settingsdomain.ClampInt(cfg.UptimeRecoveryThreshold, 1, 60)

	down := status == "down"

	var notifyDownMsg, notifyRecoveryMsg string

	s.mu.Lock()
	st := s.notifyState[siteID]
	if st == nil {
		st = &siteNotifyState{}
		s.notifyState[siteID] = st
	}
	if down {
		st.failStreak++
		st.okStreak = 0
	} else {
		st.okStreak++
		st.failStreak = 0
	}

	if !firstEver {
		if down && st.failStreak >= failTh && !st.incident && cfg.UptimeNotifyDown {
			notifyDownMsg = fmt.Sprintf("🔴 Site DOWN: %s\nError: %s", domain, checkErr)
			st.incident = true
		}
		if !down && st.incident && st.okStreak >= recoverTh && cfg.UptimeNotifyRecovery {
			notifyRecoveryMsg = fmt.Sprintf("🟢 Site RECOVERED: %s\nLatency: %dms", domain, latency)
			st.incident = false
		}
	}
	s.mu.Unlock()

	// Notifications are sent outside the mutex to avoid holding it during network I/O.
	if notifyDownMsg != "" {
		_ = s.notifier.NotifyAll(ctx, notifyDownMsg)
	}
	if notifyRecoveryMsg != "" {
		_ = s.notifier.NotifyAll(ctx, notifyRecoveryMsg)
	}
}

// PruneAll trims old uptime records for every known site. Run daily.
func (s *Service) PruneAll(ctx context.Context) {
	sites, err := s.siteRepo.List(ctx)
	if err != nil {
		return
	}
	for _, site := range sites {
		_ = s.uptimeRepo.Prune(ctx, site.ID, 10_080)
	}
}

// ping tries HTTPS first, then HTTP fallback, requesting hcPath on each.
// Returns: status ("up"/"down"), latency ms, HTTP status code, error string.
func (s *Service) ping(ctx context.Context, domain, hcPath string) (status string, latencyMs, statusCode int, errMsg string) {
	if hcPath == "" {
		hcPath = "/"
	}
	for _, scheme := range []string{"https", "http"} {
		url := scheme + "://" + domain + hcPath
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			continue
		}
		start := time.Now()
		resp, err := s.client.Do(req)
		latencyMs = int(time.Since(start).Milliseconds())
		if err != nil {
			errMsg = trimStr(err.Error(), 200)
			continue // try next scheme
		}
		resp.Body.Close()
		statusCode = resp.StatusCode
		if resp.StatusCode >= 500 {
			return "down", latencyMs, statusCode, fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
		return "up", latencyMs, statusCode, ""
	}
	return "down", latencyMs, statusCode, errMsg
}

func trimStr(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}
