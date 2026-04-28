package http

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	auditdomain "github.com/your-org/ventopanel/internal/domain/audit"
	domain "github.com/your-org/ventopanel/internal/domain/server"
	"github.com/your-org/ventopanel/internal/infra/metrics"
	provisionsvc "github.com/your-org/ventopanel/internal/service/provision"
	serversvc "github.com/your-org/ventopanel/internal/service/server"
	teamsvc "github.com/your-org/ventopanel/internal/service/team"
)

type ServerHandler struct {
	service          *serversvc.Service
	provisionService *provisionsvc.Service
	sslService       sslQueue
	teamService      *teamsvc.Service
	auditWriter      auditdomain.StatusEventWriter
}

type sslQueue interface {
	EnqueueRenew(ctx context.Context, serverID string) error
}

func NewServerHandler(
	service *serversvc.Service,
	provisionService *provisionsvc.Service,
	sslService sslQueue,
	teamService *teamsvc.Service,
	auditWriter auditdomain.StatusEventWriter,
) *ServerHandler {
	return &ServerHandler{
		service:          service,
		provisionService: provisionService,
		sslService:       sslService,
		teamService:      teamService,
		auditWriter:      auditWriter,
	}
}

func (h *ServerHandler) recordDenied(teamID, serverID, reason string) {
	metrics.IncACLDenied("server", reason)
	if h.auditWriter == nil || strings.TrimSpace(serverID) == "" {
		return
	}
	_ = h.auditWriter.WriteStatusEvent(auditdomain.StatusEvent{
		ResourceType: "server",
		ResourceID:   serverID,
		FromStatus:   "access_requested",
		ToStatus:     "access_denied",
		Reason:       reason,
		TaskID:       "acl:server:" + teamID,
	})
}

func (h *ServerHandler) authorizeServer(c *gin.Context, serverID string, requireWrite bool) bool {
	if h.teamService == nil {
		return true
	}

	teamID, ok := requireTeamID(c)
	if !ok {
		h.recordDenied("", serverID, "missing_team_identity")
		return false
	}

	if requireWrite {
		role, err := h.teamService.GetServerRole(c.Request.Context(), teamID, serverID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
			return false
		}
		switch strings.ToLower(strings.TrimSpace(role)) {
		case "owner", "admin":
			return true
		case "":
			h.recordDenied(teamID, serverID, "no_grant")
			c.JSON(http.StatusForbidden, errorResponse{Error: "forbidden"})
			return false
		default:
			h.recordDenied(teamID, serverID, "insufficient_role")
			c.JSON(http.StatusForbidden, errorResponse{Error: "forbidden: insufficient role"})
			return false
		}
	}

	allowed, err := h.teamService.HasServerAccess(c.Request.Context(), teamID, serverID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return false
	}
	if !allowed {
		h.recordDenied(teamID, serverID, "no_grant")
		c.JSON(http.StatusForbidden, errorResponse{Error: "forbidden"})
		return false
	}
	return true
}

func (h *ServerHandler) Health(c *gin.Context) {
	if err := h.service.Health(c.Request.Context()); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *ServerHandler) Create(c *gin.Context) {
	var req createServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	server, err := h.service.Create(c.Request.Context(), domain.Server{
		Name:        req.Name,
		Host:        req.Host,
		Port:        req.Port,
		Provider:    req.Provider,
		Status:      req.Status,
		SSHUser:     req.SSHUser,
		SSHPassword: req.SSHPassword,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, server)
}

func (h *ServerHandler) List(c *gin.Context) {
	if h.teamService == nil {
		servers, err := h.service.List(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
			return
		}
		c.JSON(http.StatusOK, listResponse[domain.Server]{Items: servers})
		return
	}

	teamID, ok := requireTeamID(c)
	if !ok {
		return
	}

	servers, err := h.service.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}

	filtered := make([]domain.Server, 0, len(servers))
	for _, server := range servers {
		allowed, err := h.teamService.HasServerAccess(c.Request.Context(), teamID, server.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
			return
		}
		if allowed {
			filtered = append(filtered, server)
		}
	}

	c.JSON(http.StatusOK, listResponse[domain.Server]{Items: filtered})
}

func (h *ServerHandler) GetByID(c *gin.Context) {
	if !h.authorizeServer(c, c.Param("id"), false) {
		return
	}

	server, err := h.service.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			c.JSON(http.StatusNotFound, errorResponse{Error: err.Error()})
			return
		}

		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, server)
}

func (h *ServerHandler) Update(c *gin.Context) {
	if !h.authorizeServer(c, c.Param("id"), true) {
		return
	}

	var req updateServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	server, err := h.service.Update(c.Request.Context(), domain.Server{
		ID:          c.Param("id"),
		Name:        req.Name,
		Host:        req.Host,
		Port:        req.Port,
		Provider:    req.Provider,
		Status:      req.Status,
		SSHUser:     req.SSHUser,
		SSHPassword: req.SSHPassword,
	})
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			c.JSON(http.StatusNotFound, errorResponse{Error: err.Error()})
			return
		}

		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, server)
}

func (h *ServerHandler) Delete(c *gin.Context) {
	if !h.authorizeServer(c, c.Param("id"), true) {
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

func (h *ServerHandler) Connect(c *gin.Context) {
	if !h.authorizeServer(c, c.Param("id"), true) {
		return
	}

	server, err := h.service.Connect(c.Request.Context(), c.Param("id"))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			c.JSON(http.StatusNotFound, errorResponse{Error: err.Error()})
			return
		}

		c.JSON(http.StatusBadGateway, errorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, server)
}

func (h *ServerHandler) Provision(c *gin.Context) {
	if !h.authorizeServer(c, c.Param("id"), true) {
		return
	}

	if err := h.provisionService.EnqueueProvision(c.Request.Context(), c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"status":    "queued",
		"server_id": c.Param("id"),
	})
}

func (h *ServerHandler) RenewSSL(c *gin.Context) {
	if !h.authorizeServer(c, c.Param("id"), true) {
		return
	}

	if err := h.sslService.EnqueueRenew(c.Request.Context(), c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"status":    "queued",
		"server_id": c.Param("id"),
		"task":      "ssl:renew",
	})
}

func (h *ServerHandler) GetStats(c *gin.Context) {
	if !h.authorizeServer(c, c.Param("id"), false) {
		return
	}

	stats, err := h.service.GetStats(c.Request.Context(), c.Param("id"))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			c.JSON(http.StatusNotFound, errorResponse{Error: err.Error()})
			return
		}
		c.JSON(http.StatusBadGateway, errorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}
