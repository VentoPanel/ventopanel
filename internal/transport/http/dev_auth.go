package http

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type DevAuthHandler struct {
	enabled bool
	secret  string
}

func NewDevAuthHandler(enabled bool, jwtSecret string) *DevAuthHandler {
	return &DevAuthHandler{
		enabled: enabled,
		secret:  strings.TrimSpace(jwtSecret),
	}
}

func (h *DevAuthHandler) IssueToken(c *gin.Context) {
	if !h.enabled {
		c.JSON(http.StatusNotFound, errorResponse{Error: "not found"})
		return
	}
	if h.secret == "" {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "auth secret is empty"})
		return
	}

	var req devTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	ttl := req.TTL
	if ttl <= 0 || ttl > 24*3600 {
		ttl = 3600
	}

	now := time.Now().UTC()
	claims := Claims{
		UserID: strings.TrimSpace(req.UserID),
		TeamID: strings.TrimSpace(req.TeamID),
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(ttl) * time.Second)),
			Subject:   strings.TrimSpace(req.UserID),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(h.secret))
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token": signed,
		"token_type":   "Bearer",
		"expires_in":   ttl,
	})
}
