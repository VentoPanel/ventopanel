package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func TestDevAuthHandler_IssueToken_Development(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewDevAuthHandler(true, "secret")

	engine := gin.New()
	engine.POST("/api/v1/dev/token", h.IssueToken)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/dev/token", strings.NewReader(`{"user_id":"u1","team_id":"t1","ttl_seconds":300}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if strings.TrimSpace(resp.AccessToken) == "" {
		t.Fatal("expected non-empty access_token")
	}

	claims := &Claims{}
	token, err := jwt.ParseWithClaims(resp.AccessToken, claims, func(token *jwt.Token) (any, error) {
		return []byte("secret"), nil
	})
	if err != nil || token == nil || !token.Valid {
		t.Fatalf("token should be valid, err=%v", err)
	}
	if claims.UserID != "u1" || claims.TeamID != "t1" {
		t.Fatalf("unexpected claims uid=%q tid=%q", claims.UserID, claims.TeamID)
	}
}

func TestDevAuthHandler_IssueToken_Disabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewDevAuthHandler(false, "secret")

	engine := gin.New()
	engine.POST("/api/v1/dev/token", h.IssueToken)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/dev/token", strings.NewReader(`{"user_id":"u1","team_id":"t1"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 when disabled, got %d", rec.Code)
	}
}
