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

	result, err := h.service.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidCreds) {
			c.JSON(http.StatusUnauthorized, errorResponse{Error: "invalid email or password"})
			return
		}
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}

	if result.MFARequired {
		c.JSON(http.StatusOK, gin.H{
			"mfa_required":  true,
			"mfa_session":   result.MFASession,
		})
		return
	}

	c.JSON(http.StatusOK, authResponse{
		Token: result.Token,
		Email: result.User.Email,
		Role:  result.User.Role,
	})
}

// MFAVerify handles POST /auth/mfa — second step after password login when 2FA is enabled.
func (h *AuthHandler) MFAVerify(c *gin.Context) {
	var req struct {
		MFASession string `json:"mfa_session" binding:"required"`
		Code       string `json:"code"        binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}
	token, user, err := h.service.VerifyMFA(c.Request.Context(), req.MFASession, req.Code)
	if err != nil {
		c.JSON(http.StatusUnauthorized, errorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, authResponse{Token: token, Email: user.Email, Role: user.Role})
}

// TOTPSetup handles GET /auth/totp/setup — generates a new TOTP secret (not yet active).
func (h *AuthHandler) TOTPSetup(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	secret, url, err := h.service.SetupTOTP(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"secret": secret, "url": url})
}

// TOTPEnable handles POST /auth/totp/enable — verifies the first code and activates 2FA.
// Returns a refreshed JWT so the client's token immediately reflects totp_enabled=true.
func (h *AuthHandler) TOTPEnable(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}
	if err := h.service.EnableTOTP(c.Request.Context(), userID, req.Code); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}
	token, err := h.service.IssueTokenForUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "2FA enabled", "token": token})
}

// TOTPDisable handles POST /auth/totp/disable — verifies the code and disables 2FA.
// Returns a refreshed JWT so the client's token immediately reflects totp_enabled=false.
func (h *AuthHandler) TOTPDisable(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}
	if err := h.service.DisableTOTP(c.Request.Context(), userID, req.Code); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}
	token, err := h.service.IssueTokenForUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "2FA disabled", "token": token})
}

// Register creates a new user account.
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
		"id":      user.ID,
		"email":   user.Email,
		"role":    user.Role,
		"team_id": user.TeamID,
	})
}

// requireUserID extracts the authenticated user's ID from context.
func requireUserID(c *gin.Context) (string, bool) {
	uid, exists := c.Get(contextUserIDKey)
	if !exists || uid == "" {
		c.JSON(http.StatusUnauthorized, errorResponse{Error: "authentication required"})
		return "", false
	}
	if id, ok := uid.(string); ok && id != "" {
		return id, true
	}
	c.JSON(http.StatusUnauthorized, errorResponse{Error: "authentication required"})
	return "", false
}
