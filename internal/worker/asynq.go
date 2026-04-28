package worker

import (
	"context"
	"encoding/json"

	"github.com/hibiken/asynq"

	ilogger "github.com/your-org/ventopanel/internal/infra/logger"
	alertsvc "github.com/your-org/ventopanel/internal/service/alert"
	deploysvc "github.com/your-org/ventopanel/internal/service/deploy"
	provisionsvc "github.com/your-org/ventopanel/internal/service/provision"
	sslsvc "github.com/your-org/ventopanel/internal/service/ssl"
)

type Server = asynq.Server

func NewAsynq(redisAddr string, redisDB int) (*asynq.Client, *asynq.Server) {
	opt := asynq.RedisClientOpt{
		Addr: redisAddr,
		DB:   redisDB,
	}

	client := asynq.NewClient(opt)
	server := asynq.NewServer(opt, asynq.Config{
		Concurrency: 10,
	})

	return client, server
}

func NewMux(
	logger ilogger.Logger,
	deployService *deploysvc.Service,
	provisionService *provisionsvc.Service,
	sslService *sslsvc.Service,
	alertService *alertsvc.Service,
) *asynq.ServeMux {
	mux := asynq.NewServeMux()

	mux.HandleFunc(deploysvc.TaskDeploySite, func(ctx context.Context, task *asynq.Task) error {
		var payload deploysvc.DeploySitePayload
		if err := json.Unmarshal(task.Payload(), &payload); err != nil {
			return err
		}

		logger.Info().Str("site_id", payload.SiteID).Msg("processing deploy task")

		if err := deployService.ExecuteDeploy(ctx, payload); err != nil {
			_ = alertService.NotifyAll(ctx, "site deployment failed")
			return err
		}

		return nil
	})

	mux.HandleFunc(provisionsvc.TaskProvisionServer, func(ctx context.Context, task *asynq.Task) error {
		var payload provisionsvc.ProvisionServerPayload
		if err := json.Unmarshal(task.Payload(), &payload); err != nil {
			return err
		}

		logger.Info().Str("server_id", payload.ServerID).Msg("processing provision task")

		if err := provisionService.ExecuteProvision(ctx, payload); err != nil {
			_ = alertService.NotifyAll(ctx, "server provisioning failed")
			return err
		}

		return nil
	})

	mux.HandleFunc(sslsvc.TaskIssueSSL, func(ctx context.Context, task *asynq.Task) error {
		var payload sslsvc.IssueSSLPayload
		if err := json.Unmarshal(task.Payload(), &payload); err != nil {
			return err
		}

		logger.Info().Str("site_id", payload.SiteID).Msg("processing ssl issue task")

		if err := sslService.ExecuteIssue(ctx, payload); err != nil {
			_ = alertService.NotifyAll(ctx, "ssl issuing failed")
			return err
		}

		return nil
	})

	mux.HandleFunc(sslsvc.TaskRenewSSL, func(ctx context.Context, task *asynq.Task) error {
		var payload sslsvc.RenewSSLPayload
		if err := json.Unmarshal(task.Payload(), &payload); err != nil {
			return err
		}

		logger.Info().Str("server_id", payload.ServerID).Msg("processing ssl renew task")

		if err := sslService.ExecuteRenew(ctx, payload); err != nil {
			_ = alertService.NotifyAll(ctx, "ssl renewal failed")
			return err
		}

		return nil
	})

	return mux
}
