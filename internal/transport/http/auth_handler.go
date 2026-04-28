package http

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	domain "github.com/your-org/ventopanel/internal/domain/user"
	authsvc "github.com/your-org/ventopanel/internal/service/auth"
)

type AuthHandler struct {
	service *authsvc.Service
}

func NewAuthHandler(service *authsvc.Service) *AuthHandler {
	return &AuthHandler{service: service}
}

type loginRequest struct {
	Email    string `json:"email"    binding:"required"`
	Password string `json:"password" binding:"required"`
}

type registerRequest struct {
	Email    string `json:"email"    binding:"required"`
	Password string `json:"password" binding:"required"`
	TeamID   string `json:"team_id"  binding:"required"`
}

type authResponse struct {
	Token string `json:"token"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	token, user, err := h.service.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidCreds) {
			c.JSON(http.StatusUnauthorized, errorResponse{Error: "invalid email or password"})
			return
		}
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, authResponse{
		Token: token,
		Email: user.Email,
		Role:  user.Role,
	})
}

// Register creates a new user account.
// The first user is always created as admin (bootstrap).
// Subsequent registrations require an admin JWT.
func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}
	req.Email = strings.TrimSpace(req.Email)

	user, err := h.service.Register(c.Request.Context(), req.Email, req.Password, req.TeamID)
	if err != nil {
		if errors.Is(err, domain.ErrEmailTaken) {
			c.JSON(http.StatusConflict, errorResponse{Error: err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":     user.ID,
		"email":  user.Email,
		"role":   user.Role,
		"team_id": user.TeamID,
	})
}
