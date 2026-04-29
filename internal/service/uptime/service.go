package uptime

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"

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
	incident   bool // DOWN alert already sent; waiting for recovery threshold
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

// CheckAll pings every active site and records the result.
func (s *Service) CheckAll(ctx context.Context) {
	sites, err := s.siteRepo.List(ctx)
	if err != nil {
		return
	}
	for _, site := range sites {
		switch site.Status {
		case "deployed", "ssl_pending", "ssl_active":
		default:
			continue
		}
		if strings.TrimSpace(site.Domain) == "" {
			continue
		}
		s.checkSite(ctx, site.ID, site.Domain)
	}
}

func (s *Service) checkSite(ctx context.Context, siteID, domain string) {
	url := "https://" + domain
	start := time.Now()
	resp, err := s.client.Get(url)
	if err != nil {
		url = "http://" + domain
		start = time.Now()
		resp, err = s.client.Get(url)
	}
	latency := int(time.Since(start).Milliseconds())

	check := pgrepo.UptimeCheck{SiteID: siteID}
	if err != nil {
		check.Status = "down"
		check.Error = trimError(err.Error())
	} else {
		resp.Body.Close()
		check.StatusCode = resp.StatusCode
		check.LatencyMs = latency
		if resp.StatusCode >= 500 {
			check.Status = "down"
			check.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
		} else {
			check.Status = "up"
		}
	}

	_, prevErr := s.uptimeRepo.LastCheck(ctx, siteID)

	_ = s.uptimeRepo.Insert(ctx, check)

	_ = s.uptimeRepo.Prune(ctx, siteID, 10_080)

	if prevErr != nil && prevErr != pgx.ErrNoRows {
		return
	}

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

	failTh := settingsdomain.ClampInt(cfg.UptimeFailThreshold, 1, 60)
	recoverTh := settingsdomain.ClampInt(cfg.UptimeRecoveryThreshold, 1, 60)

	down := check.Status == "down"

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

	firstEver := prevErr == pgx.ErrNoRows

	if !firstEver {
		if down && st.failStreak >= failTh && !st.incident && cfg.UptimeNotifyDown {
			notifyDownMsg = fmt.Sprintf("🔴 Site DOWN: %s\nError: %s", domain, check.Error)
			st.incident = true
		}
		if !down && st.incident && st.okStreak >= recoverTh && cfg.UptimeNotifyRecovery {
			notifyRecoveryMsg = fmt.Sprintf("🟢 Site RECOVERED: %s\nLatency: %dms", domain, check.LatencyMs)
			st.incident = false
		}
	}
	s.mu.Unlock()

	if notifyDownMsg != "" {
		_ = s.notifier.NotifyAll(ctx, notifyDownMsg)
	}
	if notifyRecoveryMsg != "" {
		_ = s.notifier.NotifyAll(ctx, notifyRecoveryMsg)
	}
}

func trimError(e string) string {
	if len(e) > 200 {
		return e[:200]
	}
	return e
}
