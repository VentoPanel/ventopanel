package http

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	sitedomain "github.com/your-org/ventopanel/internal/domain/site"
	pgrepo "github.com/your-org/ventopanel/internal/repository/postgres"
	deploysvc "github.com/your-org/ventopanel/internal/service/deploy"
)

type WebhookHandler struct {
	siteRepo      *pgrepo.SiteRepository
	deployService *deploysvc.Service
}

func NewWebhookHandler(siteRepo *pgrepo.SiteRepository, deployService *deploysvc.Service) *WebhookHandler {
	return &WebhookHandler{siteRepo: siteRepo, deployService: deployService}
}

// GenerateWebhookToken returns a cryptographically random 32-byte hex token.
func GenerateWebhookToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// Trigger POST /api/v1/webhook/:token
// Public endpoint — no JWT required. Token IS the authentication.
// Supports GitHub, GitLab, Bitbucket and plain HTTP POST.
// Branch filtering: if the payload contains a "ref" field (e.g. "refs/heads/main"),
// the deploy is skipped when the pushed branch doesn't match the site's branch.
func (h *WebhookHandler) Trigger(c *gin.Context) {
	token := strings.TrimSpace(c.Param("token"))
	if token == "" {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "missing token"})
		return
	}

	site, err := h.siteRepo.FindByWebhookToken(c.Request.Context(), token)
	if err != nil {
		if err == sitedomain.ErrNotFound {
			c.JSON(http.StatusNotFound, errorResponse{Error: "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}

	if strings.TrimSpace(site.RepositoryURL) == "" {
		c.JSON(http.StatusUnprocessableEntity, errorResponse{Error: "site has no repository URL"})
		return
	}

	// Parse optional JSON body to check the pushed branch (GitHub/GitLab send {"ref":"refs/heads/main"}).
	var payload struct {
		Ref string `json:"ref"`
	}
	// Best-effort: ignore parse errors (plain POST with no body is valid too).
	_ = c.ShouldBindJSON(&payload)

	if payload.Ref != "" {
		// "refs/heads/main" → "main", "refs/heads/feature/x" → "feature/x"
		pushedBranch := strings.TrimPrefix(payload.Ref, "refs/heads/")
		siteBranch := site.Branch
		if siteBranch == "" {
			siteBranch = "main"
		}
		if pushedBranch != siteBranch {
			c.JSON(http.StatusOK, gin.H{
				"ok":      false,
				"message": "branch not tracked, deploy skipped",
				"pushed":  pushedBranch,
				"tracked": siteBranch,
			})
			return
		}
	}

	if err := h.deployService.EnqueueDeploy(c.Request.Context(), site.ID); err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"ok":      true,
		"site_id": site.ID,
		"message": "deploy queued",
	})
}

// Regenerate POST /sites/:id/webhook/regenerate (authenticated)
// Issues a new token, invalidating the old one.
func (h *WebhookHandler) Regenerate(c *gin.Context) {
	siteID := c.Param("id")

	token, err := GenerateWebhookToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "failed to generate token"})
		return
	}

	if err := h.siteRepo.UpdateWebhookToken(c.Request.Context(), siteID, token); err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"webhook_token": token})
}
