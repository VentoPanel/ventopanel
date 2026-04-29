package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	pgrepo "github.com/your-org/ventopanel/internal/repository/postgres"
)

type APITokenHandler struct {
	repo *pgrepo.APITokenRepository
}

func NewAPITokenHandler(repo *pgrepo.APITokenRepository) *APITokenHandler {
	return &APITokenHandler{repo: repo}
}

// ListTokens handles GET /api-tokens — returns all tokens for the current user.
func (h *APITokenHandler) ListTokens(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	tokens, err := h.repo.ListByUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}

	type item struct {
		ID          string  `json:"id"`
		Name        string  `json:"name"`
		LastUsedAt  *string `json:"last_used_at"`
		CreatedAt   string  `json:"created_at"`
	}
	out := make([]item, 0, len(tokens))
	for _, t := range tokens {
		it := item{
			ID:        t.ID,
			Name:      t.Name,
			CreatedAt: t.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
		if t.LastUsedAt != nil {
			s := t.LastUsedAt.Format("2006-01-02T15:04:05Z")
			it.LastUsedAt = &s
		}
		out = append(out, it)
	}
	c.JSON(http.StatusOK, gin.H{"items": out})
}

// CreateToken handles POST /api-tokens — generates a new token and returns the plaintext ONCE.
func (h *APITokenHandler) CreateToken(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "name is required"})
		return
	}

	plaintext, hash, err := pgrepo.GenerateToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "could not generate token"})
		return
	}

	t, err := h.repo.Create(c.Request.Context(), userID, req.Name, hash)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":        t.ID,
		"name":      t.Name,
		"token":     plaintext, // shown ONCE — client must copy immediately
		"created_at": t.CreatedAt.Format("2006-01-02T15:04:05Z"),
	})
}

// RevokeToken handles DELETE /api-tokens/:id — deletes the token (user owns it).
func (h *APITokenHandler) RevokeToken(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	if err := h.repo.Delete(c.Request.Context(), c.Param("id"), userID); err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
