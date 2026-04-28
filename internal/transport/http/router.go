package http

import "github.com/gin-gonic/gin"

func RegisterRoutes(
	engine *gin.Engine,
	healthHandler *HealthHandler,
	metricsHandler *MetricsHandler,
	devAuthHandler *DevAuthHandler,
	authHandler *AuthHandler,
	serverHandler *ServerHandler,
	siteHandler *SiteHandler,
	teamHandler *TeamHandler,
	observabilityHandler *ObservabilityHandler,
	auditHandler *AuditHandler,
	settingsHandler *SettingsHandler,
	userHandler *UserHandler,
) {
	engine.GET("/metrics", metricsHandler.Get)

	api := engine.Group("/api/v1")
	{
		api.GET("/health", healthHandler.Get)
		api.POST("/auth/login", authHandler.Login)
		api.POST("/auth/register", authHandler.Register)
		if devAuthHandler != nil {
			api.POST("/dev/token", devAuthHandler.IssueToken)
		}
		api.POST("/servers", serverHandler.Create)
		api.GET("/servers", serverHandler.List)
		api.GET("/servers/:id", serverHandler.GetByID)
		api.PUT("/servers/:id", serverHandler.Update)
		api.DELETE("/servers/:id", serverHandler.Delete)
		api.POST("/servers/:id/connect", serverHandler.Connect)
		api.POST("/servers/:id/provision", serverHandler.Provision)
		api.POST("/servers/:id/ssl/renew", serverHandler.RenewSSL)
		api.GET("/servers/:id/stats", serverHandler.GetStats)
		api.GET("/servers/health", serverHandler.Health)
		api.POST("/sites", siteHandler.Create)
		api.GET("/sites", siteHandler.List)
		api.GET("/sites/:id", siteHandler.GetByID)
		api.PUT("/sites/:id", siteHandler.Update)
		api.DELETE("/sites/:id", siteHandler.Delete)
		api.POST("/sites/:id/deploy", siteHandler.Deploy)
		api.GET("/sites/:id/logs", siteHandler.GetLogs)
		api.GET("/teams/access", teamHandler.List)
		api.GET("/observability/ssl", observabilityHandler.SSL)
		api.GET("/audit/status-events", auditHandler.ListStatusEvents)
		api.GET("/settings/notifications", settingsHandler.GetNotifications)
		api.PATCH("/settings/notifications", settingsHandler.UpdateNotifications)
		api.GET("/users", userHandler.List)
		api.PATCH("/users/:id/role", userHandler.UpdateRole)
		api.DELETE("/users/:id", userHandler.Delete)
	}
}
