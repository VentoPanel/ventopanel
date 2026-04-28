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
	"github.com/your-org/ventopanel/internal/infra/db"
	icrypto "github.com/your-org/ventopanel/internal/infra/crypto"
	ilock "github.com/your-org/ventopanel/internal/infra/lock"
	ilogger "github.com/your-org/ventopanel/internal/infra/logger"
	"github.com/your-org/ventopanel/internal/infra/notifier"
	"github.com/your-org/ventopanel/internal/infra/security"
	"github.com/your-org/ventopanel/internal/infra/ssh"
	postgresrepo "github.com/your-org/ventopanel/internal/repository/postgres"
	alertsvc "github.com/your-org/ventopanel/internal/service/alert"
	auditsvc "github.com/your-org/ventopanel/internal/service/audit"
	deploysvc "github.com/your-org/ventopanel/internal/service/deploy"
	provisionsvc "github.com/your-org/ventopanel/internal/service/provision"
	serversvc "github.com/your-org/ventopanel/internal/service/server"
	sitesvc "github.com/your-org/ventopanel/internal/service/site"
	sslsvc "github.com/your-org/ventopanel/internal/service/ssl"
	teamsvc "github.com/your-org/ventopanel/internal/service/team"
	httptransport "github.com/your-org/ventopanel/internal/transport/http"
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
	teamRepo := postgresrepo.NewTeamRepository(pgPool)
	statusEventRepo := postgresrepo.NewStatusEventRepository(pgPool)

	sshExecutor := ssh.NewExecutor(cfg.SSHDialTimeout)
	lockManager := ilock.NewRedisLockManager(redisClient)
	firewallManager := security.NewFirewallManager()
	sslManager := security.NewSSLManager(sshExecutor, cfg.SSLCertbotEmail)
	telegramNotifier := notifier.NewTelegram(cfg.TelegramBotToken, cfg.TelegramChatID)
	whatsAppNotifier := notifier.NewWhatsApp(cfg.WhatsAppWebhookURL)

	serverService := serversvc.NewService(serverRepo, sshExecutor, statusEventRepo)
	siteService := sitesvc.NewService(siteRepo, serverRepo)
	teamService := teamsvc.NewService(teamRepo)
	auditService := auditsvc.NewService(statusEventRepo)
	sslService := sslsvc.NewService(siteRepo, serverRepo, sslManager, asynqClient, lockManager, statusEventRepo)
	deployService := deploysvc.NewService(siteRepo, serverRepo, sshExecutor, firewallManager, sslManager, sslService, asynqClient, lockManager, statusEventRepo)
	provisionService := provisionsvc.NewService(serverRepo, sshExecutor, asynqClient, lockManager, statusEventRepo)
	alertService := alertsvc.NewService(telegramNotifier, whatsAppNotifier)

	engine := buildRouter(cfg, logger, serverService, siteService, teamService, deployService, provisionService, sslService, auditService)
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
	serverService *serversvc.Service,
	siteService *sitesvc.Service,
	teamService *teamsvc.Service,
	deployService *deploysvc.Service,
	provisionService *provisionsvc.Service,
	sslService *sslsvc.Service,
	auditService *auditsvc.Service,
) *gin.Engine {
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(httptransport.RequestIDMiddleware())
	engine.Use(httptransport.LoggerMiddleware(logger))
	engine.Use(httptransport.AuthContextMiddleware(cfg.AuthJWTSecret, cfg.AuthAllowHeaders))

	healthHandler := httptransport.NewHealthHandler()
	metricsHandler := httptransport.NewMetricsHandler()
	devAuthHandler := httptransport.NewDevAuthHandler(cfg.AppEnv == "development", cfg.AuthJWTSecret)
	serverHandler := httptransport.NewServerHandler(serverService, provisionService, sslService, teamService)
	siteHandler := httptransport.NewSiteHandler(siteService, deployService, teamService)
	teamHandler := httptransport.NewTeamHandler(teamService)
	observabilityHandler := httptransport.NewObservabilityHandler(sslService)
	auditHandler := httptransport.NewAuditHandler(auditService)

	httptransport.RegisterRoutes(engine, healthHandler, metricsHandler, devAuthHandler, serverHandler, siteHandler, teamHandler, observabilityHandler, auditHandler)

	return engine
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
