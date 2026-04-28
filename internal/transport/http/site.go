package http

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	domain "github.com/your-org/ventopanel/internal/domain/site"
	deploysvc "github.com/your-org/ventopanel/internal/service/deploy"
	sitesvc "github.com/your-org/ventopanel/internal/service/site"
	teamsvc "github.com/your-org/ventopanel/internal/service/team"
)

type SiteHandler struct {
	service       *sitesvc.Service
	deployService *deploysvc.Service
	teamService   *teamsvc.Service
}

func (h *SiteHandler) authorizeSite(c *gin.Context, siteID string, requireWrite bool) bool {
	teamID, ok := requireTeamID(c)
	if !ok {
		return false
	}

	if requireWrite {
		role, err := h.teamService.GetSiteRole(c.Request.Context(), teamID, siteID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
			return false
		}
		switch strings.ToLower(strings.TrimSpace(role)) {
		case "owner", "admin":
			return true
		case "":
			c.JSON(http.StatusForbidden, errorResponse{Error: "forbidden"})
			return false
		default:
			c.JSON(http.StatusForbidden, errorResponse{Error: "forbidden: insufficient role"})
			return false
		}
	}

	allowed, err := h.teamService.HasSiteAccess(c.Request.Context(), teamID, siteID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return false
	}
	if !allowed {
		c.JSON(http.StatusForbidden, errorResponse{Error: "forbidden"})
		return false
	}
	return true
}

func NewSiteHandler(service *sitesvc.Service, deployService *deploysvc.Service, teamService *teamsvc.Service) *SiteHandler {
	return &SiteHandler{service: service, deployService: deployService, teamService: teamService}
}

func (h *SiteHandler) Create(c *gin.Context) {
	var req createSiteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	site, err := h.service.Create(c.Request.Context(), domain.Site{
		ServerID:      req.ServerID,
		Name:          req.Name,
		Domain:        req.Domain,
		Runtime:       req.Runtime,
		RepositoryURL: req.RepositoryURL,
		Status:        req.Status,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrServerNotFound):
			c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		}
		return
	}

	c.JSON(http.StatusCreated, site)
}

func (h *SiteHandler) List(c *gin.Context) {
	sites, err := h.service.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, listResponse[domain.Site]{Items: sites})
}

func (h *SiteHandler) GetByID(c *gin.Context) {
	if !h.authorizeSite(c, c.Param("id"), false) {
		return
	}

	site, err := h.service.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			c.JSON(http.StatusNotFound, errorResponse{Error: err.Error()})
			return
		}

		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, site)
}

func (h *SiteHandler) Update(c *gin.Context) {
	if !h.authorizeSite(c, c.Param("id"), true) {
		return
	}

	var req updateSiteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	site, err := h.service.Update(c.Request.Context(), domain.Site{
		ID:            c.Param("id"),
		ServerID:      req.ServerID,
		Name:          req.Name,
		Domain:        req.Domain,
		Runtime:       req.Runtime,
		RepositoryURL: req.RepositoryURL,
		Status:        req.Status,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNotFound):
			c.JSON(http.StatusNotFound, errorResponse{Error: err.Error()})
		case errors.Is(err, domain.ErrServerNotFound):
			c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, site)
}

func (h *SiteHandler) Delete(c *gin.Context) {
	if !h.authorizeSite(c, c.Param("id"), true) {
		return
	}

	err := h.service.Delete(c.Request.Context(), c.Param("id"))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			c.JSON(http.StatusNotFound, errorResponse{Error: err.Error()})
			return
		}

		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *SiteHandler) Deploy(c *gin.Context) {
	if !h.authorizeSite(c, c.Param("id"), true) {
		return
	}

	_ = h.service

	if err := h.deployService.EnqueueDeploy(c.Request.Context(), c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"status":  "queued",
		"site_id": c.Param("id"),
	})
}
