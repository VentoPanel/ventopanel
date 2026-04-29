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
	envHandler *EnvHandler,
	webhookHandler *WebhookHandler,
	uptimeHandler *UptimeHandler,
	backupHandler *BackupHandler,
	dashboardHandler *DashboardHandler,
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
		api.GET("/servers/:id/sites", serverHandler.GetServerSites)
		api.GET("/servers/:id/containers", serverHandler.GetServerContainers)
		api.GET("/servers/health", serverHandler.Health)
		api.POST("/sites", siteHandler.Create)
		api.GET("/sites", siteHandler.List)
		api.GET("/sites/:id", siteHandler.GetByID)
		api.PUT("/sites/:id", siteHandler.Update)
		api.DELETE("/sites/:id", siteHandler.Delete)
		api.POST("/sites/:id/deploy", siteHandler.Deploy)
		api.GET("/sites/:id/ssl", siteHandler.GetSSLInfo)
		api.POST("/sites/:id/ssl/renew", siteHandler.RenewSSL)
		api.GET("/sites/:id/logs", siteHandler.GetLogs)
		api.GET("/sites/:id/container", siteHandler.GetContainerInfo)
		api.GET("/sites/:id/container/logs", siteHandler.GetContainerLogs)
		api.POST("/sites/:id/container/restart", siteHandler.RestartContainer)
		api.GET("/sites/:id/env", envHandler.ListEnv)
		api.PUT("/sites/:id/env", envHandler.UpsertEnv)
		api.DELETE("/sites/:id/env/:key", envHandler.DeleteEnv)
		api.POST("/sites/:id/webhook/regenerate", webhookHandler.Regenerate)
		api.GET("/sites/:id/uptime", uptimeHandler.GetUptime)
		api.GET("/uptime/overview", uptimeHandler.GetOverview)
		api.GET("/backups", backupHandler.ListBackups)
		api.POST("/backups/trigger", backupHandler.TriggerBackup)
		api.GET("/backups/:name/download", backupHandler.DownloadBackup)
		// Public — no JWT, token IS the auth.
		api.POST("/webhook/:token", webhookHandler.Trigger)
		api.GET("/teams/access", teamHandler.List)
		api.GET("/observability/ssl", observabilityHandler.SSL)
		// Both paths resolve to the same handler for backward compatibility.
		api.GET("/audit", auditHandler.ListStatusEvents)
		api.GET("/audit/status-events", auditHandler.ListStatusEvents)
		api.GET("/settings/notifications", settingsHandler.GetNotifications)
		api.PATCH("/settings/notifications", settingsHandler.UpdateNotifications)
		api.GET("/users", userHandler.List)
		api.PATCH("/users/:id/role", userHandler.UpdateRole)
		api.DELETE("/users/:id", userHandler.Delete)
		api.GET("/dashboard/summary", dashboardHandler.GetSummary)
		api.GET("/dashboard/uptime-trend", dashboardHandler.GetUptimeTrend)
		api.GET("/dashboard/deploy-trend", dashboardHandler.GetDeployTrend)
	}
}
