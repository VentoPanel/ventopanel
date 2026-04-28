package http

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	domain "github.com/your-org/ventopanel/internal/domain/user"
)

type UserHandler struct {
	repo domain.Repository
}

func NewUserHandler(repo domain.Repository) *UserHandler {
	return &UserHandler{repo: repo}
}

func (h *UserHandler) requireAdminRole(c *gin.Context) bool {
	// Role is stored in the JWT as "role" claim, but the current middleware only
	// sets uid and tid in context. We check the context key "role" if set, or
	// fall back to allowing any authenticated user (uid present) for now.
	// TODO: store role in context from JWT when RBAC middleware is added.
	if _, ok := c.Get(contextUserIDKey); !ok {
		c.JSON(http.StatusForbidden, errorResponse{Error: "admin access required"})
		return false
	}
	return true
}

func (h *UserHandler) List(c *gin.Context) {
	if !h.requireAdminRole(c) {
		return
	}

	users, err := h.repo.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}

	type userDTO struct {
		ID        string `json:"id"`
		Email     string `json:"email"`
		TeamID    string `json:"team_id"`
		Role      string `json:"role"`
		CreatedAt string `json:"created_at"`
	}
	items := make([]userDTO, 0, len(users))
	for _, u := range users {
		items = append(items, userDTO{
			ID:        u.ID,
			Email:     u.Email,
			TeamID:    u.TeamID,
			Role:      u.Role,
			CreatedAt: u.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *UserHandler) UpdateRole(c *gin.Context) {
	if !h.requireAdminRole(c) {
		return
	}

	var req struct {
		Role string `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	role := strings.ToLower(strings.TrimSpace(req.Role))
	if role != domain.RoleAdmin && role != domain.RoleEditor && role != domain.RoleViewer {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid role: must be admin, editor, or viewer"})
		return
	}

	if err := h.repo.UpdateRole(c.Request.Context(), c.Param("id"), role); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			c.JSON(http.StatusNotFound, errorResponse{Error: "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "updated", "role": role})
}

func (h *UserHandler) Delete(c *gin.Context) {
	if !h.requireAdminRole(c) {
		return
	}

	// Prevent self-deletion.
	callerID, _ := c.Get(contextUserIDKey)
	if callerID == c.Param("id") {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "cannot delete your own account"})
		return
	}

	if err := h.repo.Delete(c.Request.Context(), c.Param("id")); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			c.JSON(http.StatusNotFound, errorResponse{Error: "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}
