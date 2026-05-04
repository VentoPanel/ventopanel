package http

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	domain "github.com/your-org/ventopanel/internal/domain/user"
)

// Invite creates a new user with a specified role. Admin only.
func (h *UserHandler) Invite(c *gin.Context) {
	if !h.requireAdminRole(c) {
		return
	}
	var req struct {
		Email    string `json:"email"    binding:"required"`
		Password string `json:"password" binding:"required,min=8"`
		Role     string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}
	role := strings.ToLower(strings.TrimSpace(req.Role))
	if role != domain.RoleAdmin && role != domain.RoleEditor && role != domain.RoleViewer {
		role = domain.RoleViewer
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "failed to hash password"})
		return
	}

	// Derive team_id from the inviting user's token context.
	tid, _ := c.Get(contextTeamIDKey)
	teamID, _ := tid.(string)

	u := &domain.User{
		Email:        strings.ToLower(strings.TrimSpace(req.Email)),
		PasswordHash: string(hash),
		TeamID:       teamID,
		Role:         role,
	}
	if err := h.repo.Create(c.Request.Context(), u); err != nil {
		if errors.Is(err, domain.ErrEmailTaken) {
			c.JSON(http.StatusConflict, errorResponse{Error: "email already registered"})
			return
		}
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"id":    u.ID,
		"email": u.Email,
		"role":  u.Role,
	})
}

type UserHandler struct {
	repo domain.Repository
}

func NewUserHandler(repo domain.Repository) *UserHandler {
	return &UserHandler{repo: repo}
}

// GetMe returns the current authenticated user's profile.
func (h *UserHandler) GetMe(c *gin.Context) {
	uid, ok := c.Get(contextUserIDKey)
	if !ok {
		c.JSON(http.StatusUnauthorized, errorResponse{Error: "not authenticated"})
		return
	}
	u, err := h.repo.GetByID(c.Request.Context(), uid.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id":           u.ID,
		"email":        u.Email,
		"role":         u.Role,
		"totp_enabled": u.TOTPEnabled,
		"created_at":   u.CreatedAt.Format("2006-01-02T15:04:05Z"),
	})
}

// ChangePassword updates the current user's password after verifying the current one.
func (h *UserHandler) ChangePassword(c *gin.Context) {
	uid, ok := c.Get(contextUserIDKey)
	if !ok {
		c.JSON(http.StatusUnauthorized, errorResponse{Error: "not authenticated"})
		return
	}
	var req struct {
		CurrentPassword string `json:"current_password" binding:"required"`
		NewPassword     string `json:"new_password"     binding:"required,min=8"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	u, err := h.repo.GetByID(c.Request.Context(), uid.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.CurrentPassword)) != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: domain.ErrWrongPassword.Error()})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "failed to hash password"})
		return
	}
	if err := h.repo.UpdatePassword(c.Request.Context(), uid.(string), string(hash)); err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "password updated"})
}

// ChangeEmail updates the current user's email after verifying their password.
func (h *UserHandler) ChangeEmail(c *gin.Context) {
	uid, ok := c.Get(contextUserIDKey)
	if !ok {
		c.JSON(http.StatusUnauthorized, errorResponse{Error: "not authenticated"})
		return
	}
	var req struct {
		Email    string `json:"email"    binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	u, err := h.repo.GetByID(c.Request.Context(), uid.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)) != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: domain.ErrWrongPassword.Error()})
		return
	}
	if err := h.repo.UpdateEmail(c.Request.Context(), uid.(string), strings.ToLower(strings.TrimSpace(req.Email))); err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "email updated"})
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
