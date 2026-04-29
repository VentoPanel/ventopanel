package http

import (
	"errors"
	"fmt"
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

	webhookToken, _ := GenerateWebhookToken()
	branch := strings.TrimSpace(req.Branch)
	if branch == "" {
		branch = "main"
	}
	hcPath := strings.TrimSpace(req.HealthcheckPath)
	if hcPath == "" {
		hcPath = "/"
	}
	site, err := h.service.Create(c.Request.Context(), domain.Site{
		ServerID:        req.ServerID,
		Name:            req.Name,
		Domain:          req.Domain,
		Runtime:         req.Runtime,
		RepositoryURL:   req.RepositoryURL,
		Branch:          branch,
		Status:          req.Status,
		WebhookToken:    webhookToken,
		HealthcheckPath: hcPath,
		TemplateID:      strings.TrimSpace(req.TemplateID),
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

	updBranch := strings.TrimSpace(req.Branch)
	if updBranch == "" {
		updBranch = "main"
	}
	updHcPath := strings.TrimSpace(req.HealthcheckPath)
	if updHcPath == "" {
		updHcPath = "/"
	}
	site, err := h.service.Update(c.Request.Context(), domain.Site{
		ID:              c.Param("id"),
		ServerID:        req.ServerID,
		Name:            req.Name,
		Domain:          req.Domain,
		Runtime:         req.Runtime,
		RepositoryURL:   req.RepositoryURL,
		Branch:          updBranch,
		Status:          req.Status,
		HealthcheckPath: updHcPath,
		TemplateID:      strings.TrimSpace(req.TemplateID),
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

// StreamContainerLogs handles GET /sites/:id/container/logs/stream
// It streams live Docker container logs as Server-Sent Events.
func (h *SiteHandler) StreamContainerLogs(c *gin.Context) {
	id := c.Param("id")
	if !h.authorizeSite(c, id, false) {
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no") // disable Nginx buffering

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "streaming unsupported"})
		return
	}

	// sseWriter wraps gin's ResponseWriter and formats each chunk as an SSE event.
	sseWriter := &sseLineWriter{w: c.Writer, flusher: flusher}

	// Use the request context — it's cancelled when the client disconnects.
	_ = h.deployService.StreamContainerLogs(c.Request.Context(), id, sseWriter)

	// Send a closing event so the frontend knows the stream ended.
	fmt.Fprintf(c.Writer, "event: close\ndata: stream ended\n\n")
	flusher.Flush()
}

// sseLineWriter formats writes as SSE "data:" lines.
type sseLineWriter struct {
	w       gin.ResponseWriter
	flusher http.Flusher
	buf     strings.Builder
}

func (s *sseLineWriter) Write(p []byte) (int, error) {
	s.buf.Write(p)
	// Flush complete lines as SSE events.
	for {
		content := s.buf.String()
		idx := strings.IndexByte(content, '\n')
		if idx < 0 {
			break
		}
		line := strings.TrimRight(content[:idx], "\r")
		s.buf.Reset()
		s.buf.WriteString(content[idx+1:])
		fmt.Fprintf(s.w, "data: %s\n\n", line)
		s.flusher.Flush()
	}
	return len(p), nil
}

// GetCommits handles GET /sites/:id/commits — returns recent git commits on the server.
func (h *SiteHandler) GetCommits(c *gin.Context) {
	id := c.Param("id")
	if !h.authorizeSite(c, id, false) {
		return
	}
	commits, err := h.deployService.GetCommits(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": commits})
}

// Rollback handles POST /sites/:id/rollback — checks out a commit and rebuilds.
func (h *SiteHandler) Rollback(c *gin.Context) {
	id := c.Param("id")
	if !h.authorizeSite(c, id, true) {
		return
	}
	var req struct {
		Commit string `json:"commit" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}
	if err := h.deployService.RollbackToCommit(c.Request.Context(), id, req.Commit); err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// GetDeployHistory handles GET /sites/:id/deploys?limit=20
func (h *SiteHandler) GetDeployHistory(c *gin.Context) {
	id := c.Param("id")
	if !h.authorizeSite(c, id, false) {
		return
	}
	limit := 20
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	logs, err := h.taskLogRepo.ListBySiteID(c.Request.Context(), id, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}
	type deployLogJSON struct {
		ID         string  `json:"id"`
		TaskType   string  `json:"task_type"`
		Status     string  `json:"status"`
		Output     string  `json:"output"`
		StartedAt  string  `json:"started_at"`
		FinishedAt *string `json:"finished_at"`
		DurationMs *int64  `json:"duration_ms"`
	}
	out := make([]deployLogJSON, 0, len(logs))
	for _, l := range logs {
		row := deployLogJSON{
			ID:        l.ID,
			TaskType:  l.TaskType,
			Status:    l.Status,
			Output:    l.Output,
			StartedAt: l.StartedAt.Format("2006-01-02T15:04:05Z"),
		}
		if l.FinishedAt != nil {
			s := l.FinishedAt.Format("2006-01-02T15:04:05Z")
			row.FinishedAt = &s
			ms := l.FinishedAt.Sub(l.StartedAt).Milliseconds()
			row.DurationMs = &ms
		}
		out = append(out, row)
	}
	c.JSON(http.StatusOK, gin.H{"items": out})
}
