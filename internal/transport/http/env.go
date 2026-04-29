package http

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"

	pgrepo "github.com/your-org/ventopanel/internal/repository/postgres"
	teamsvc "github.com/your-org/ventopanel/internal/service/team"
)

var validEnvKey = regexp.MustCompile(`^[A-Z_][A-Z0-9_]{0,127}$`)

type EnvHandler struct {
	repo        *pgrepo.EnvRepository
	teamService *teamsvc.Service
}

func NewEnvHandler(repo *pgrepo.EnvRepository, ts *teamsvc.Service) *EnvHandler {
	return &EnvHandler{repo: repo, teamService: ts}
}

func (h *EnvHandler) authorize(c *gin.Context, siteID string, write bool) bool {
	teamID, ok := requireTeamID(c)
	if !ok {
		return false
	}
	if write {
		role, err := h.teamService.GetSiteRole(c.Request.Context(), teamID, siteID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
			return false
		}
		switch strings.ToLower(strings.TrimSpace(role)) {
		case "owner", "admin", "editor":
			return true
		default:
			c.JSON(http.StatusForbidden, errorResponse{Error: "forbidden"})
			return false
		}
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

// ListEnv GET /sites/:id/env
func (h *EnvHandler) ListEnv(c *gin.Context) {
	id := c.Param("id")
	if !h.authorize(c, id, false) {
		return
	}
	vars, err := h.repo.ListBySiteID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}
	type item struct {
		Key       string `json:"key"`
		Value     string `json:"value"`
		UpdatedAt string `json:"updated_at"`
	}
	out := make([]item, 0, len(vars))
	for _, v := range vars {
		out = append(out, item{Key: v.Key, Value: v.Value, UpdatedAt: v.UpdatedAt.Format("2006-01-02T15:04:05Z")})
	}
	c.JSON(http.StatusOK, gin.H{"items": out})
}

// UpsertEnv PUT /sites/:id/env
// Body: { "key": "FOO", "value": "bar" }
func (h *EnvHandler) UpsertEnv(c *gin.Context) {
	id := c.Param("id")
	if !h.authorize(c, id, true) {
		return
	}
	var body struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid JSON"})
		return
	}
	body.Key = strings.TrimSpace(strings.ToUpper(body.Key))
	if !validEnvKey.MatchString(body.Key) {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "key must match [A-Z_][A-Z0-9_]{0,127}"})
		return
	}
	if err := h.repo.Upsert(c.Request.Context(), id, body.Key, body.Value); err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "key": body.Key})
}

// DeleteEnv DELETE /sites/:id/env/:key
func (h *EnvHandler) DeleteEnv(c *gin.Context) {
	id := c.Param("id")
	if !h.authorize(c, id, true) {
		return
	}
	key := strings.ToUpper(strings.TrimSpace(c.Param("key")))
	if err := h.repo.Delete(c.Request.Context(), id, key); err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
