package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/your-org/ventopanel/internal/app/config"
	icrypto "github.com/your-org/ventopanel/internal/infra/crypto"
	"github.com/your-org/ventopanel/internal/infra/db"
	ilock "github.com/your-org/ventopanel/internal/infra/lock"
	ilogger "github.com/your-org/ventopanel/internal/infra/logger"
	"github.com/your-org/ventopanel/internal/infra/security"
	"github.com/your-org/ventopanel/internal/infra/ssh"
	postgresrepo "github.com/your-org/ventopanel/internal/repository/postgres"
	alertsvc "github.com/your-org/ventopanel/internal/service/alert"
	backupsvc "github.com/your-org/ventopanel/internal/service/backup"
	uptimesvc "github.com/your-org/ventopanel/internal/service/uptime"
	authsvc "github.com/your-org/ventopanel/internal/service/auth"
	auditsvc "github.com/your-org/ventopanel/internal/service/audit"
	deploysvc "github.com/your-org/ventopanel/internal/service/deploy"
	provisionsvc "github.com/your-org/ventopanel/internal/service/provision"
	serversvc "github.com/your-org/ventopanel/internal/service/server"
	sitesvc "github.com/your-org/ventopanel/internal/service/site"
	sslsvc "github.com/your-org/ventopanel/internal/service/ssl"
	teamsvc "github.com/your-org/ventopanel/internal/service/team"
	httptransport "github.com/your-org/ventopanel/internal/transport/http"
	settingsdomain "github.com/your-org/ventopanel/internal/domain/settings"
	"github.com/your-org/ventopanel/internal/worker"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	logger := ilogger.New(cfg.LogLevel)

	ctx := context.Background()

	pgPool, err := db.NewPostgres(ctx, cfg.PostgresDSN)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to connect to postgres")
	}
	defer pgPool.Close()

	if err := db.RunMigrations(cfg.PostgresDSN, "migrations"); err != nil {
		logger.Fatal().Err(err).Msg("failed to run database migrations")
	}
	logger.Info().Msg("database migrations applied")

	redisClient := db.NewRedis(cfg.RedisAddr, cfg.RedisDB)
	defer func() {
		if err := redisClient.Close(); err != nil {
			logger.Error().Err(err).Msg("failed to close redis client")
		}
	}()

	asynqClient, asynqServer := worker.NewAsynq(cfg.RedisAddr, cfg.RedisDB)
	defer func() {
		if err := asynqClient.Close(); err != nil {
			logger.Error().Err(err).Msg("failed to close asynq client")
		}
	}()

	encryptor, err := icrypto.NewEncryptor(cfg.AppEncryptionKey)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to initialize encryption")
	}

	serverRepo := postgresrepo.NewServerRepository(pgPool, encryptor)
	siteRepo := postgresrepo.NewSiteRepository(pgPool)
	envRepo := postgresrepo.NewEnvRepository(pgPool, encryptor)
	uptimeRepo := postgresrepo.NewUptimeRepository(pgPool)
	teamRepo := postgresrepo.NewTeamRepository(pgPool)
	userRepo := postgresrepo.NewUserRepository(pgPool)
	statusEventRepo := postgresrepo.NewStatusEventRepository(pgPool)
	taskLogRepo := postgresrepo.NewTaskLogRepository(pgPool)
	settingsRepo := postgresrepo.NewSettingsRepository(pgPool)

	// Seed DB with env-based notifier config if DB values are still empty.
	if cfg.TelegramBotToken != "" || cfg.TelegramChatID != "" || cfg.WhatsAppWebhookURL != "" {
		existing, _ := settingsRepo.GetNotificationConfig(ctx)
		if existing.TelegramBotToken == "" && existing.TelegramChatID == "" && existing.WhatsAppWebhookURL == "" {
			_ = settingsRepo.SetNotificationConfig(ctx, settingsdomain.NotificationConfig{
				TelegramBotToken:        cfg.TelegramBotToken,
				TelegramChatID:          cfg.TelegramChatID,
				WhatsAppWebhookURL:      cfg.WhatsAppWebhookURL,
				UptimeNotifyDown:        true,
				UptimeNotifyRecovery:    true,
				UptimeFailThreshold:     1,
				UptimeRecoveryThreshold: 1,
			})
			logger.Info().Msg("seeded notification settings from environment variables")
		}
	}

	sshExecutor := ssh.NewExecutor(cfg.SSHDialTimeout)
	lockManager := ilock.NewRedisLockManager(redisClient)
	firewallManager := security.NewFirewallManager()
	sslManager := security.NewSSLManager(sshExecutor, cfg.SSLCertbotEmail)
	authService := authsvc.NewService(
		userRepo,
		cfg.AuthJWTSecret, cfg.AuthJWTIssuer, cfg.AuthJWTAudience,
		12*time.Hour,
	)
	serverService := serversvc.NewService(serverRepo, sshExecutor, statusEventRepo)
	siteService := sitesvc.NewService(siteRepo, serverRepo)
	teamService := teamsvc.NewService(teamRepo)
	auditService := auditsvc.NewService(statusEventRepo)
	sslService := sslsvc.NewService(siteRepo, serverRepo, sslManager, asynqClient, lockManager, statusEventRepo).WithSSH(sshExecutor)
	deployService := deploysvc.NewService(siteRepo, serverRepo, sshExecutor, firewallManager, sslManager, sslService, asynqClient, lockManager, statusEventRepo, taskLogRepo, envRepo)
	provisionService := provisionsvc.NewService(serverRepo, sshExecutor, asynqClient, lockManager, statusEventRepo)
	// Alert service reads config dynamically from DB on every send — no static notifiers needed.
	alertService := alertsvc.NewService().WithSettingsRepo(settingsRepo)
	uptimeService := uptimesvc.NewService(siteRepo, uptimeRepo, alertService, settingsRepo)
	backupService := backupsvc.NewService(pgPool, cfg.BackupDir, cfg.BackupKeepCount, alertService)
	dashboardRepo := postgresrepo.NewDashboardRepository(pgPool)

	engine := buildRouter(cfg, logger, authService, serverService, siteService, teamService, deployService, provisionService, sslService, auditService, statusEventRepo, taskLogRepo, settingsRepo, userRepo, envRepo, siteRepo, uptimeRepo, backupService, dashboardRepo)
	httpServer := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           engine,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	workerMux := worker.NewMux(logger, deployService, provisionService, sslService, alertService)
	schedulerCtx, schedulerCancel := context.WithCancel(context.Background())
	startSSLRenewScheduler(schedulerCtx, logger, sslService)
	startUptimeScheduler(schedulerCtx, logger, uptimeService)
	startBackupScheduler(schedulerCtx, logger, backupService, alertService)

	go func() {
		logger.Info().Str("addr", httpServer.Addr).Str("env", cfg.AppEnv).Msg("starting http server")
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal().Err(err).Msg("http server stopped with error")
		}
	}()

	go func() {
		logger.Info().Msg("starting asynq worker")
		if err := asynqServer.Run(workerMux); err != nil {
			logger.Fatal().Err(err).Msg("worker stopped with error")
		}
	}()

	waitForShutdown(logger, httpServer, asynqServer, schedulerCancel)
}

func buildRouter(
	cfg *config.Config,
	logger ilogger.Logger,
	authService *authsvc.Service,
	serverService *serversvc.Service,
	siteService *sitesvc.Service,
	teamService *teamsvc.Service,
	deployService *deploysvc.Service,
	provisionService *provisionsvc.Service,
	sslService *sslsvc.Service,
	auditService *auditsvc.Service,
	statusEventRepo *postgresrepo.StatusEventRepository,
	taskLogRepo *postgresrepo.TaskLogRepository,
	settingsRepo *postgresrepo.SettingsRepository,
	userRepo *postgresrepo.UserRepository,
	envRepo *postgresrepo.EnvRepository,
	siteRepo *postgresrepo.SiteRepository,
	uptimeRepo *postgresrepo.UptimeRepository,
	backupService *backupsvc.Service,
	dashboardRepo *postgresrepo.DashboardRepository,
) *gin.Engine {
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(httptransport.RequestIDMiddleware())
	engine.Use(httptransport.LoggerMiddleware(logger))
	engine.Use(httptransport.AuthContextMiddlewareWithOptions(httptransport.AuthOptions{
		JWTSecret:           cfg.AuthJWTSecret,
		AllowHeaderFallback: cfg.AuthAllowHeaders,
		ExpectedIssuer:      cfg.AuthJWTIssuer,
		ExpectedAudience:    cfg.AuthJWTAudience,
	}))

	healthHandler := httptransport.NewHealthHandler()
	metricsHandler := httptransport.NewMetricsHandler()
	devAuthHandler := httptransport.NewDevAuthHandler(cfg.AppEnv == "development", cfg.AuthJWTSecret)
	authHandler := httptransport.NewAuthHandler(authService)
	serverHandler := httptransport.NewServerHandler(serverService, provisionService, sslService, teamService, statusEventRepo, deployService, siteRepo)
	siteHandler := httptransport.NewSiteHandler(siteService, deployService, teamService, statusEventRepo, taskLogRepo, sslService)
	teamHandler := httptransport.NewTeamHandler(teamService)
	observabilityHandler := httptransport.NewObservabilityHandler(sslService)
	auditHandler := httptransport.NewAuditHandler(auditService)
	settingsHandler := httptransport.NewSettingsHandler(settingsRepo)
	userHandler := httptransport.NewUserHandler(userRepo)
	envHandler := httptransport.NewEnvHandler(envRepo, teamService)
	webhookHandler := httptransport.NewWebhookHandler(siteRepo, deployService)
	uptimeHandler := httptransport.NewUptimeHandler(uptimeRepo, teamService)
	backupHandler := httptransport.NewBackupHandler(backupService)
	dashboardHandler := httptransport.NewDashboardHandler(dashboardRepo)
	templateHandler := httptransport.NewTemplateHandler()

	httptransport.RegisterRoutes(engine, healthHandler, metricsHandler, devAuthHandler, authHandler, serverHandler, siteHandler, teamHandler, observabilityHandler, auditHandler, settingsHandler, userHandler, envHandler, webhookHandler, uptimeHandler, backupHandler, dashboardHandler, templateHandler)

	return engine
}

type alertNotifier interface {
	NotifyAll(ctx context.Context, message string) error
}

func startBackupScheduler(ctx context.Context, logger ilogger.Logger, svc *backupsvc.Service, notifier alertNotifier) {
	go func() {
		// First backup 5 min after startup to verify the setup.
		select {
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Minute):
		}
		runBackup(ctx, logger, svc, notifier)

		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				runBackup(ctx, logger, svc, notifier)
			}
		}
	}()
	logger.Info().Msg("backup scheduler started (interval: 24h)")
}

func runBackup(ctx context.Context, logger ilogger.Logger, svc *backupsvc.Service, notifier alertNotifier) {
	if err := svc.Run(ctx); err != nil {
		logger.Error().Err(err).Msg("scheduled backup failed")
		_ = notifier.NotifyAll(ctx, "⚠️ VentoPanel backup FAILED\n"+err.Error())
	} else {
		logger.Info().Msg("scheduled backup completed")
	}
}

func startUptimeScheduler(ctx context.Context, logger ilogger.Logger, svc *uptimesvc.Service) {
	// Ping goroutine: every 60s.
	go func() {
		select {
		case <-ctx.Done():
			return
		case <-time.After(30 * time.Second):
		}
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		for {
			svc.CheckAll(ctx)
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()

	// Prune goroutine: once per day per site — avoids per-check DB cost.
	go func() {
		select {
		case <-ctx.Done():
			return
		case <-time.After(2 * time.Minute): // let first check cycle finish first
		}
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			svc.PruneAll(ctx)
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()

	logger.Info().Msg("uptime monitor started (check: 60s, prune: 24h)")
}

func startSSLRenewScheduler(ctx context.Context, logger ilogger.Logger, sslService *sslsvc.Service) {
	go func() {
		// Run once shortly after startup, then once a day.
		initialDelay := time.Minute
		select {
		case <-ctx.Done():
			return
		case <-time.After(initialDelay):
		}

		if err := sslService.EnqueueDailyRenewForAll(ctx, 6*time.Hour); err != nil {
			logger.Error().Err(err).Msg("failed to enqueue initial ssl renew batch")
		}

		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := sslService.EnqueueDailyRenewForAll(ctx, 6*time.Hour); err != nil {
					logger.Error().Err(err).Msg("failed to enqueue daily ssl renew batch")
				}
			}
		}
	}()
}

func waitForShutdown(
	logger ilogger.Logger,
	httpServer *http.Server,
	asynqServer *worker.Server,
	stopSchedulers context.CancelFunc,
) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	sig := <-stop
	logger.Info().Str("signal", sig.String()).Msg("shutdown signal received")

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error().Err(err).Msg("failed to shutdown http server gracefully")
	} else {
		logger.Info().Msg("http server stopped")
	}

	stopSchedulers()
	asynqServer.Shutdown()
	logger.Info().Msg("worker stopped")
	logger.Info().Msg("application shutdown completed")
}
