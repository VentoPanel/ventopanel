package http

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	pgrepo "github.com/your-org/ventopanel/internal/repository/postgres"
)

const (
	contextUserIDKey = "auth_user_id"
	contextTeamIDKey = "auth_team_id"
	contextRoleKey   = "auth_role"
)

type Claims struct {
	UserID  string `json:"uid"`
	TeamID  string `json:"tid"`
	Role    string `json:"role"`
	// TeamIDLegacy supports tokens issued with "team_id" claim (older tooling / scripts).
	TeamIDLegacy string `json:"team_id"`
	jwt.RegisteredClaims
}

// apiTokenLookup is an interface so the middleware can resolve API tokens without
// depending directly on the concrete repo type in every call site.
type apiTokenLookup interface {
	GetByHash(ctx context.Context, hash string) (*pgrepo.APIToken, error)
	TouchLastUsed(ctx context.Context, id string)
}

type AuthOptions struct {
	JWTSecret           string
	AllowHeaderFallback bool
	ExpectedIssuer      string
	ExpectedAudience    string
	APITokenRepo        apiTokenLookup // optional; enables Bearer vp_... API tokens
}

func AuthContextMiddleware(jwtSecret string, allowHeaderFallback bool) gin.HandlerFunc {
	return AuthContextMiddlewareWithOptions(AuthOptions{
		JWTSecret:           jwtSecret,
		AllowHeaderFallback: allowHeaderFallback,
	})
}

func AuthContextMiddlewareWithOptions(opts AuthOptions) gin.HandlerFunc {
	secret := strings.TrimSpace(opts.JWTSecret)
	expectedIssuer := strings.TrimSpace(opts.ExpectedIssuer)
	expectedAudience := strings.TrimSpace(opts.ExpectedAudience)

	return func(c *gin.Context) {
		tokenString := strings.TrimSpace(strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer "))
		// Allow token via query param for SSE/EventSource which cannot set headers.
		if tokenString == "" {
			tokenString = strings.TrimSpace(c.Query("token"))
		}

		if tokenString != "" {
			// Try JWT first.
			if secret != "" {
				claims := &Claims{}
				parseOpts := make([]jwt.ParserOption, 0, 2)
				if expectedIssuer != "" {
					parseOpts = append(parseOpts, jwt.WithIssuer(expectedIssuer))
				}
				if expectedAudience != "" {
					parseOpts = append(parseOpts, jwt.WithAudience(expectedAudience))
				}
				tok, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
					return []byte(secret), nil
				}, parseOpts...)
				if err == nil && tok != nil && tok.Valid {
					if uid := strings.TrimSpace(claims.UserID); uid != "" {
						c.Set(contextUserIDKey, uid)
					}
					tid := strings.TrimSpace(claims.TeamID)
					if tid == "" {
						tid = strings.TrimSpace(claims.TeamIDLegacy)
					}
					if tid != "" {
						c.Set(contextTeamIDKey, tid)
					}
					if role := strings.TrimSpace(claims.Role); role != "" {
						c.Set(contextRoleKey, role)
					}
				}
			}

			// If JWT didn't authenticate and token looks like an API token (vp_…), try DB lookup.
			if _, authenticated := c.Get(contextUserIDKey); !authenticated &&
				opts.APITokenRepo != nil &&
				strings.HasPrefix(tokenString, "vp_") {

				hash := pgrepo.HashToken(tokenString)
				if apiTok, err := opts.APITokenRepo.GetByHash(c.Request.Context(), hash); err == nil && apiTok != nil {
					c.Set(contextUserIDKey, apiTok.UserID)
					// API tokens get admin role for full access; adjust if per-token roles are needed.
					c.Set(contextRoleKey, "admin")
					go opts.APITokenRepo.TouchLastUsed(context.Background(), apiTok.ID)
				}
			}
		}

		if opts.AllowHeaderFallback {
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
