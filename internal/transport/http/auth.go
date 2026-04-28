package http

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const (
	contextUserIDKey = "auth_user_id"
	contextTeamIDKey = "auth_team_id"
)

type Claims struct {
	UserID string `json:"uid"`
	TeamID string `json:"tid"`
	jwt.RegisteredClaims
}

func AuthContextMiddleware(jwtSecret string, allowHeaderFallback bool) gin.HandlerFunc {
	secret := strings.TrimSpace(jwtSecret)

	return func(c *gin.Context) {
		tokenString := strings.TrimSpace(strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer "))
		if tokenString != "" && secret != "" {
			claims := &Claims{}
			token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
				return []byte(secret), nil
			})
			if err == nil && token != nil && token.Valid {
				if uid := strings.TrimSpace(claims.UserID); uid != "" {
					c.Set(contextUserIDKey, uid)
				}
				if tid := strings.TrimSpace(claims.TeamID); tid != "" {
					c.Set(contextTeamIDKey, tid)
				}
			}
		}

		if allowHeaderFallback {
			if _, ok := c.Get(contextUserIDKey); !ok {
				if v := strings.TrimSpace(c.GetHeader("X-User-ID")); v != "" {
					c.Set(contextUserIDKey, v)
				}
			}
			if _, ok := c.Get(contextTeamIDKey); !ok {
				if v := strings.TrimSpace(c.GetHeader("X-Team-ID")); v != "" {
					c.Set(contextTeamIDKey, v)
				}
			}
		}

		c.Next()
	}
}

func TeamIDFromRequest(c *gin.Context) (string, bool) {
	if v, ok := c.Get(contextTeamIDKey); ok {
		if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s), true
		}
	}
	return "", false
}

func requireTeamID(c *gin.Context) (string, bool) {
	teamID, ok := TeamIDFromRequest(c)
	if !ok {
		c.JSON(http.StatusForbidden, errorResponse{Error: "forbidden: missing team identity"})
		return "", false
	}
	return teamID, true
}
