package http

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	pgrepo "github.com/your-org/ventopanel/internal/repository/postgres"
	teamsvc "github.com/your-org/ventopanel/internal/service/team"
)

type UptimeHandler struct {
	uptimeRepo  *pgrepo.UptimeRepository
	teamService *teamsvc.Service
}

func NewUptimeHandler(
	uptimeRepo *pgrepo.UptimeRepository,
	teamService *teamsvc.Service,
) *UptimeHandler {
	return &UptimeHandler{uptimeRepo: uptimeRepo, teamService: teamService}
}

type uptimeCheckJSON struct {
	ID         string `json:"id"`
	CheckedAt  string `json:"checked_at"`
	Status     string `json:"status"`
	LatencyMs  int    `json:"latency_ms"`
	StatusCode int    `json:"status_code,omitempty"`
	Error      string `json:"error,omitempty"`
}

type uptimeResponse struct {
	UptimePct float64           `json:"uptime_pct"`
	Checks    []uptimeCheckJSON `json:"checks"`
}

type uptimeSiteOverviewJSON struct {
	SiteID        string  `json:"site_id"`
	SiteName      string  `json:"site_name"`
	Domain        string  `json:"domain"`
	LastStatus    string  `json:"last_status"`
	LastCheckedAt string  `json:"last_checked_at"`
	LatencyMs     int     `json:"latency_ms"`
	UptimePct90   float64 `json:"uptime_pct_90"`
}

// GetOverview handles GET /uptime/overview — all sites with latest status and uptime %.
func (h *UptimeHandler) GetOverview(c *gin.Context) {
	overviews, err := h.uptimeRepo.OverviewAll(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}

	out := make([]uptimeSiteOverviewJSON, 0, len(overviews))
	for _, o := range overviews {
		out = append(out, uptimeSiteOverviewJSON{
			SiteID:        o.SiteID,
			SiteName:      o.SiteName,
			Domain:        o.Domain,
			LastStatus:    o.LastStatus,
			LastCheckedAt: o.LastCheckedAt.UTC().Format("2006-01-02T15:04:05Z"),
			LatencyMs:     o.LatencyMs,
			UptimePct90:   o.UptimePct90,
		})
	}

	c.JSON(http.StatusOK, gin.H{"items": out})
}

// GetUptime handles GET /sites/:id/uptime?limit=90
func (h *UptimeHandler) GetUptime(c *gin.Context) {
	siteID := c.Param("id")

	limit := 90
	if l := c.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 1440 {
			limit = v
		}
	}

	// ACL: caller must have access to this site.
	teamID, ok := requireTeamID(c)
	if !ok {
		return
	}
	allowed, err := h.teamService.HasSiteAccess(c.Request.Context(), teamID, siteID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}
	if !allowed {
		c.JSON(http.StatusForbidden, errorResponse{Error: "forbidden"})
		return
	}

	checks, err := h.uptimeRepo.ListRecent(c.Request.Context(), siteID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}

	pct, _ := h.uptimeRepo.UptimePct(c.Request.Context(), siteID, limit)

	resp := uptimeResponse{
		UptimePct: pct,
		Checks:    make([]uptimeCheckJSON, 0, len(checks)),
	}
	for _, ch := range checks {
		resp.Checks = append(resp.Checks, uptimeCheckJSON{
			ID:         ch.ID,
			CheckedAt:  ch.CheckedAt.UTC().Format("2006-01-02T15:04:05Z"),
			Status:     ch.Status,
			LatencyMs:  ch.LatencyMs,
			StatusCode: ch.StatusCode,
			Error:      ch.Error,
		})
	}

	c.JSON(http.StatusOK, resp)
}
