package http

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	teamsvc "github.com/your-org/ventopanel/internal/service/team"
)

type siteDomainRepository interface {
	List(ctx context.Context, siteID string) ([]string, error)
	Add(ctx context.Context, siteID, domain string) error
	Remove(ctx context.Context, siteID, domain string) error
}

type SiteDomainHandler struct {
	repo        siteDomainRepository
	teamService *teamsvc.Service
}

func NewSiteDomainHandler(repo siteDomainRepository, ts *teamsvc.Service) *SiteDomainHandler {
	return &SiteDomainHandler{repo: repo, teamService: ts}
}

func (h *SiteDomainHandler) authorizeWrite(c *gin.Context, siteID string) bool {
	teamID, ok := requireTeamID(c)
	if !ok {
		return false
	}
	role, err := h.teamService.GetSiteRole(c.Request.Context(), teamID, siteID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return false
	}
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "owner", "admin", "editor":
		return true
	default:
		c.JSON(http.StatusForbidden, errorResponse{Error: "write access required"})
		return false
	}
}

func (h *SiteDomainHandler) authorizeRead(c *gin.Context, siteID string) bool {
	teamID, ok := requireTeamID(c)
	if !ok {
		return false
	}
	ok2, err := h.teamService.HasSiteAccess(c.Request.Context(), teamID, siteID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return false
	}
	if !ok2 {
		c.JSON(http.StatusForbidden, errorResponse{Error: "forbidden"})
		return false
	}
	return true
}

// ListDomains handles GET /sites/:id/domains
func (h *SiteDomainHandler) ListDomains(c *gin.Context) {
	siteID := c.Param("id")
	if !h.authorizeRead(c, siteID) {
		return
	}
	domains, err := h.repo.List(c.Request.Context(), siteID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}
	if domains == nil {
		domains = []string{}
	}
	c.JSON(http.StatusOK, gin.H{"items": domains})
}

// AddDomain handles POST /sites/:id/domains
// Body: {"domain": "alias.example.com"}
func (h *SiteDomainHandler) AddDomain(c *gin.Context) {
	siteID := c.Param("id")
	if !h.authorizeWrite(c, siteID) {
		return
	}
	var req struct {
		Domain string `json:"domain" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "domain is required"})
		return
	}
	domain := strings.ToLower(strings.TrimSpace(req.Domain))
	if domain == "" {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "domain must not be empty"})
		return
	}
	if err := h.repo.Add(c.Request.Context(), siteID, domain); err != nil {
		c.JSON(http.StatusConflict, errorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"domain": domain})
}

// RemoveDomain handles DELETE /sites/:id/domains/:domain
func (h *SiteDomainHandler) RemoveDomain(c *gin.Context) {
	siteID := c.Param("id")
	if !h.authorizeWrite(c, siteID) {
		return
	}
	domain := c.Param("domain")
	if err := h.repo.Remove(c.Request.Context(), siteID, domain); err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
