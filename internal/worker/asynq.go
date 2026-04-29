package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

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
			logger.Error().Str("site_id", payload.SiteID).Err(err).Msg("deploy task failed")
			// Don't retry on invalid lifecycle transitions — the site is already
			// in a terminal state (e.g. ssl_pending → deploying is not allowed).
			if errors.Is(err, asynq.SkipRetry) {
				return err
			}
			_ = alertService.NotifyAll(ctx, fmt.Sprintf(
				"🚨 <b>Site deploy FAILED</b>\n"+
					"Site: <code>%s</code>\n"+
					"Error: <code>%s</code>",
				payload.SiteID, err.Error(),
			))
			return err
		}

		_ = alertService.NotifyAll(ctx, fmt.Sprintf(
			"✅ <b>Site deployed successfully</b>\n"+
				"Site: <code>%s</code>",
			payload.SiteID,
		))
		return nil
	})

	mux.HandleFunc(provisionsvc.TaskProvisionServer, func(ctx context.Context, task *asynq.Task) error {
		var payload provisionsvc.ProvisionServerPayload
		if err := json.Unmarshal(task.Payload(), &payload); err != nil {
			return err
		}

		logger.Info().Str("server_id", payload.ServerID).Msg("processing provision task")

		if err := provisionService.ExecuteProvision(ctx, payload); err != nil {
			logger.Error().Str("server_id", payload.ServerID).Err(err).Msg("provision task failed")
			_ = alertService.NotifyAll(ctx, fmt.Sprintf(
				"🚨 <b>Server provisioning FAILED</b>\n"+
					"Server: <code>%s</code>\n"+
					"Error: <code>%s</code>",
				payload.ServerID, err.Error(),
			))
			return err
		}

		_ = alertService.NotifyAll(ctx, fmt.Sprintf(
			"✅ <b>Server provisioned successfully</b>\n"+
				"Server: <code>%s</code>",
			payload.ServerID,
		))
		return nil
	})

	mux.HandleFunc(sslsvc.TaskIssueSSL, func(ctx context.Context, task *asynq.Task) error {
		var payload sslsvc.IssueSSLPayload
		if err := json.Unmarshal(task.Payload(), &payload); err != nil {
			return err
		}

		logger.Info().Str("site_id", payload.SiteID).Msg("processing ssl issue task")

		if err := sslService.ExecuteIssue(ctx, payload); err != nil {
			_ = alertService.NotifyAll(ctx, fmt.Sprintf(
				"🚨 <b>SSL issue FAILED</b>\n"+
					"Site: <code>%s</code>\n"+
					"Error: <code>%s</code>",
				payload.SiteID, err.Error(),
			))
			return err
		}

		_ = alertService.NotifyAll(ctx, fmt.Sprintf(
			"🔒 <b>SSL certificate issued</b>\n"+
				"Site: <code>%s</code>",
			payload.SiteID,
		))
		return nil
	})

	mux.HandleFunc(sslsvc.TaskRenewSSL, func(ctx context.Context, task *asynq.Task) error {
		var payload sslsvc.RenewSSLPayload
		if err := json.Unmarshal(task.Payload(), &payload); err != nil {
			return err
		}

		logger.Info().Str("server_id", payload.ServerID).Msg("processing ssl renew task")

		if err := sslService.ExecuteRenew(ctx, payload); err != nil {
			_ = alertService.NotifyAll(ctx, fmt.Sprintf(
				"🚨 <b>SSL renewal FAILED</b>\n"+
					"Server: <code>%s</code>\n"+
					"Error: <code>%s</code>",
				payload.ServerID, err.Error(),
			))
			return err
		}

		_ = alertService.NotifyAll(ctx, fmt.Sprintf(
			"🔒 <b>SSL certificates renewed</b>\n"+
				"Server: <code>%s</code>",
			payload.ServerID,
		))
		return nil
	})

	return mux
}
