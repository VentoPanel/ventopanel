package http

import "github.com/gin-gonic/gin"

func RegisterRoutes(
	engine *gin.Engine,
	healthHandler *HealthHandler,
	metricsHandler *MetricsHandler,
	devAuthHandler *DevAuthHandler,
	serverHandler *ServerHandler,
	siteHandler *SiteHandler,
	teamHandler *TeamHandler,
	observabilityHandler *ObservabilityHandler,
	auditHandler *AuditHandler,
) {
	engine.GET("/metrics", metricsHandler.Get)

	api := engine.Group("/api/v1")
	{
		api.GET("/health", healthHandler.Get)
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
		api.GET("/servers/health", serverHandler.Health)
		api.POST("/sites", siteHandler.Create)
		api.GET("/sites", siteHandler.List)
		api.GET("/sites/:id", siteHandler.GetByID)
		api.PUT("/sites/:id", siteHandler.Update)
		api.DELETE("/sites/:id", siteHandler.Delete)
		api.POST("/sites/:id/deploy", siteHandler.Deploy)
		api.GET("/teams/access", teamHandler.List)
		api.GET("/observability/ssl", observabilityHandler.SSL)
		api.GET("/audit/status-events", auditHandler.ListStatusEvents)
	}
}
