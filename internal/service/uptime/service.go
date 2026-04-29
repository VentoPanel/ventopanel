package uptime

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	pgrepo "github.com/your-org/ventopanel/internal/repository/postgres"
)

type notifier interface {
	NotifyAll(ctx context.Context, message string) error
}

type Service struct {
	siteRepo   *pgrepo.SiteRepository
	uptimeRepo *pgrepo.UptimeRepository
	notifier   notifier
	client     *http.Client
}

func NewService(
	siteRepo *pgrepo.SiteRepository,
	uptimeRepo *pgrepo.UptimeRepository,
	notifier notifier,
) *Service {
	return &Service{
		siteRepo:   siteRepo,
		uptimeRepo: uptimeRepo,
		notifier:   notifier,
		client: &http.Client{
			Timeout: 10 * time.Second,
			// Follow redirects (HTTP→HTTPS) — count any response as "up".
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
// It is designed to be called from a scheduler every minute.
func (s *Service) CheckAll(ctx context.Context) {
	sites, err := s.siteRepo.List(ctx)
	if err != nil {
		return
	}
	for _, site := range sites {
		// Only monitor sites that are expected to be serving traffic.
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
	// Try HTTPS first; HTTP as fallback.
	url := "https://" + domain
	start := time.Now()
	resp, err := s.client.Get(url)
	if err != nil {
		// Fallback to HTTP.
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
		// Treat any HTTP response (including 4xx) as "up" — the server answered.
		// 5xx and network errors are "down".
		if resp.StatusCode >= 500 {
			check.Status = "down"
			check.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
		} else {
			check.Status = "up"
		}
	}

	// Load previous status for change detection before inserting.
	prev, prevErr := s.uptimeRepo.LastCheck(ctx, siteID)

	_ = s.uptimeRepo.Insert(ctx, check)

	// Prune old records to keep table bounded (~7 days at 1 check/min).
	_ = s.uptimeRepo.Prune(ctx, siteID, 10_080)

	// Send notification on state change.
	if prevErr != nil && prevErr != pgx.ErrNoRows {
		return
	}
	if prev == nil {
		return // First ever check — no notification.
	}
	if prev.Status == check.Status {
		return // No change.
	}

	var msg string
	if check.Status == "down" {
		msg = fmt.Sprintf("🔴 Site DOWN: %s\nError: %s", domain, check.Error)
	} else {
		msg = fmt.Sprintf("🟢 Site RECOVERED: %s\nLatency: %dms", domain, check.LatencyMs)
	}
	_ = s.notifier.NotifyAll(ctx, msg)
}

func trimError(e string) string {
	if len(e) > 200 {
		return e[:200]
	}
	return e
}
