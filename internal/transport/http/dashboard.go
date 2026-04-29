package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	pgrepo "github.com/your-org/ventopanel/internal/repository/postgres"
)

// DashboardHandler serves aggregated platform metrics for the observability page.
type DashboardHandler struct {
	repo *pgrepo.DashboardRepository
}

func NewDashboardHandler(repo *pgrepo.DashboardRepository) *DashboardHandler {
	return &DashboardHandler{repo: repo}
}

// GetSummary handles GET /dashboard/summary
// Returns counts for sites, servers, uptime, and recent deploys in one call.
func (h *DashboardHandler) GetSummary(c *gin.Context) {
	ctx := c.Request.Context()

	sites, err := h.repo.GetSiteSummary(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}
	servers, err := h.repo.GetServerSummary(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}
	uptime, err := h.repo.GetUptimeSummary(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}
	deploys, err := h.repo.GetDeploySummary(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"sites":   sites,
		"servers": servers,
		"uptime":  uptime,
		"deploys": deploys,
	})
}

// GetUptimeTrend handles GET /dashboard/uptime-trend
// Returns hourly uptime check counts for the last 24 hours.
func (h *DashboardHandler) GetUptimeTrend(c *gin.Context) {
	points, err := h.repo.GetUptimeTrend(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"points": points})
}

// GetDeployTrend handles GET /dashboard/deploy-trend
// Returns daily deploy success/failure counts for the last 7 days.
func (h *DashboardHandler) GetDeployTrend(c *gin.Context) {
	points, err := h.repo.GetDeployTrend(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"points": points})
}
