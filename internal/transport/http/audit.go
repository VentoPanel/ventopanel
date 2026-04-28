package http

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	auditsvc "github.com/your-org/ventopanel/internal/service/audit"
)

type AuditHandler struct {
	service *auditsvc.Service
}

func NewAuditHandler(service *auditsvc.Service) *AuditHandler {
	return &AuditHandler{service: service}
}

func (h *AuditHandler) ListStatusEvents(c *gin.Context) {
	var since *time.Time
	if raw := strings.TrimSpace(c.Query("since")); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid since; use RFC3339"})
			return
		}
		since = &parsed
	}

	limit := 100
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid limit"})
			return
		}
		limit = v
	}

	var beforeCreatedAt *time.Time
	var beforeID string
	if raw := strings.TrimSpace(c.Query("before")); raw != "" {
		parts := strings.SplitN(raw, ",", 2)
		if len(parts) != 2 {
			c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid before; use <RFC3339>,<event_id>"})
			return
		}

		parsed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(parts[0]))
		if err != nil {
			c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid before timestamp; use RFC3339"})
			return
		}
		beforeCreatedAt = &parsed
		beforeID = strings.TrimSpace(parts[1])
	}

	includeTotal := false
	if raw := strings.TrimSpace(c.Query("include_total")); raw != "" {
		v, err := strconv.ParseBool(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid include_total; use true|false"})
			return
		}
		includeTotal = v
	}

	result, err := h.service.List(auditsvc.Filter{
		ResourceType: strings.TrimSpace(c.Query("resource_type")),
		ResourceID:   strings.TrimSpace(c.Query("resource_id")),
		FromStatus:   strings.TrimSpace(c.Query("from")),
		ToStatus:     strings.TrimSpace(c.Query("to")),
		Since:        since,
		BeforeCreatedAt: beforeCreatedAt,
		BeforeID:     beforeID,
		Limit:        limit,
		IncludeTotal: includeTotal,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}

	var nextCursor string
	if len(result.Items) == limit {
		last := result.Items[len(result.Items)-1]
		nextCursor = last.CreatedAt.UTC().Format(time.RFC3339Nano) + "," + last.ID
	}

	response := gin.H{
		"items":       result.Items,
		"next_cursor": nextCursor,
	}
	if result.TotalCount != nil {
		response["total_count"] = *result.TotalCount
	}

	c.JSON(http.StatusOK, response)
}
