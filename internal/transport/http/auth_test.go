package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func TestAuthContextMiddleware_JWTClaims(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(AuthContextMiddleware("secret", false))
	engine.GET("/me", func(c *gin.Context) {
		teamID, ok := TeamIDFromRequest(c)
		if !ok {
			c.Status(http.StatusForbidden)
			return
		}
		c.String(http.StatusOK, teamID)
	})

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		UserID: "user-1",
		TeamID: "team-1",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	})
	tokenStr, err := token.SignedString([]byte("secret"))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestAuthContextMiddleware_HeaderFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(AuthContextMiddleware("", true))
	engine.GET("/me", func(c *gin.Context) {
		teamID, ok := TeamIDFromRequest(c)
		if !ok || teamID != "team-header" {
			c.Status(http.StatusForbidden)
			return
		}
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("X-Team-ID", "team-header")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}
