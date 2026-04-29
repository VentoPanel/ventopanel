package http

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	auditdomain "github.com/your-org/ventopanel/internal/domain/audit"
	domain "github.com/your-org/ventopanel/internal/domain/site"
	"github.com/your-org/ventopanel/internal/domain/tasklog"
	"github.com/your-org/ventopanel/internal/infra/metrics"
	deploysvc "github.com/your-org/ventopanel/internal/service/deploy"
	sitesvc "github.com/your-org/ventopanel/internal/service/site"
	sslsvc "github.com/your-org/ventopanel/internal/service/ssl"
	teamsvc "github.com/your-org/ventopanel/internal/service/team"
)

type SiteHandler struct {
	service       *sitesvc.Service
	deployService *deploysvc.Service
	teamService   *teamsvc.Service
	auditWriter   auditdomain.StatusEventWriter
	taskLogRepo   tasklog.Repository
	sslService    *sslsvc.Service
}

func (h *SiteHandler) recordDenied(teamID, siteID, reason string) {
	metrics.IncACLDenied("site", reason)
	if h.auditWriter == nil || strings.TrimSpace(siteID) == "" {
		return
	}
	_ = h.auditWriter.WriteStatusEvent(auditdomain.StatusEvent{
		ResourceType: "site",
		ResourceID:   siteID,
		FromStatus:   "access_requested",
		ToStatus:     "access_denied",
		Reason:       reason,
		TaskID:       "acl:site:" + teamID,
	})
}

func (h *SiteHandler) authorizeSite(c *gin.Context, siteID string, requireWrite bool) bool {
	teamID, ok := requireTeamID(c)
	if !ok {
		h.recordDenied("", siteID, "missing_team_identity")
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
			h.recordDenied(teamID, siteID, "no_grant")
			c.JSON(http.StatusForbidden, errorResponse{Error: "forbidden"})
			return false
		default:
			h.recordDenied(teamID, siteID, "insufficient_role")
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
		h.recordDenied(teamID, siteID, "no_grant")
		c.JSON(http.StatusForbidden, errorResponse{Error: "forbidden"})
		return false
	}
	return true
}

func NewSiteHandler(
	service *sitesvc.Service,
	deployService *deploysvc.Service,
	teamService *teamsvc.Service,
	auditWriter auditdomain.StatusEventWriter,
	taskLogRepo tasklog.Repository,
	sslService *sslsvc.Service,
) *SiteHandler {
	return &SiteHandler{
		service:       service,
		deployService: deployService,
		teamService:   teamService,
		auditWriter:   auditWriter,
		taskLogRepo:   taskLogRepo,
		sslService:    sslService,
	}
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

	// Grant the creating team owner access so the site appears in their list.
	if teamID, ok := TeamIDFromRequest(c); ok && teamID != "" {
		_ = h.teamService.GrantSiteAccess(c.Request.Context(), teamID, site.ID, "owner")
	}

	c.JSON(http.StatusCreated, site)
}

func (h *SiteHandler) List(c *gin.Context) {
	if h.teamService == nil {
		sites, err := h.service.List(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
			return
		}
		c.JSON(http.StatusOK, listResponse[domain.Site]{Items: sites})
		return
	}

	teamID, ok := requireTeamID(c)
	if !ok {
		return
	}

	sites, err := h.service.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}

	filtered := make([]domain.Site, 0, len(sites))
	for _, site := range sites {
		allowed, err := h.teamService.HasSiteAccess(c.Request.Context(), teamID, site.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
			return
		}
		if allowed {
			filtered = append(filtered, site)
		}
	}

	c.JSON(http.StatusOK, listResponse[domain.Site]{Items: filtered})
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

func (h *SiteHandler) GetSSLInfo(c *gin.Context) {
	if !h.authorizeSite(c, c.Param("id"), false) {
		return
	}

	if h.sslService == nil {
		c.JSON(http.StatusOK, gin.H{"status": "no_cert"})
		return
	}

	info, err := h.sslService.GetCertInfo(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, info)
}

func (h *SiteHandler) RenewSSL(c *gin.Context) {
	if !h.authorizeSite(c, c.Param("id"), true) {
		return
	}

	if h.sslService == nil {
		c.JSON(http.StatusServiceUnavailable, errorResponse{Error: "ssl service not configured"})
		return
	}

	if err := h.sslService.EnqueueIssue(c.Request.Context(), c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"status": "queued", "site_id": c.Param("id")})
}

func (h *SiteHandler) GetLogs(c *gin.Context) {
	if !h.authorizeSite(c, c.Param("id"), false) {
		return
	}

	limit := 20
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}

	if h.taskLogRepo == nil {
		c.JSON(http.StatusOK, gin.H{"items": []struct{}{}})
		return
	}

	logs, err := h.taskLogRepo.ListBySiteID(c.Request.Context(), c.Param("id"), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}
	if logs == nil {
		logs = []tasklog.TaskLog{}
	}
	c.JSON(http.StatusOK, gin.H{"items": logs})
}

func (h *SiteHandler) GetContainerInfo(c *gin.Context) {
	id := c.Param("id")
	if !h.authorizeSite(c, id, false) {
		return
	}
	info, err := h.deployService.GetContainerInfo(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, info)
}

func (h *SiteHandler) GetContainerLogs(c *gin.Context) {
	id := c.Param("id")
	if !h.authorizeSite(c, id, false) {
		return
	}
	tail := 100
	if v := c.Query("tail"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 500 {
			tail = n
		}
	}
	out, err := h.deployService.GetContainerLogs(c.Request.Context(), id, tail)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"logs": out})
}

func (h *SiteHandler) RestartContainer(c *gin.Context) {
	id := c.Param("id")
	if !h.authorizeSite(c, id, true) {
		return
	}
	if err := h.deployService.RestartContainer(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
