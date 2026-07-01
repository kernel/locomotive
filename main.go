package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/brody192/locomotive/internal/config"
	"github.com/brody192/locomotive/internal/errgroup"
	"github.com/brody192/locomotive/internal/logger"
	"github.com/brody192/locomotive/internal/logline/serializer"
	"github.com/brody192/locomotive/internal/otel"
	"github.com/brody192/locomotive/internal/pipeline"
	"github.com/brody192/locomotive/internal/railway"
	"github.com/brody192/locomotive/internal/railway/subscribe/environment_logs"
	"github.com/brody192/locomotive/internal/railway/subscribe/http_logs"
)

func main() {
	os.Exit(run())
}

func run() int {
	logger.Stdout.Info("Preparing the locomotive for departure...")

	gqlClient, err := railway.NewClient(
		railway.AuthToken(config.Global.RailwayApiKey),
		railway.BaseURL("https://backboard.railway.app/graphql/v2"),
		railway.BaseSubscriptionURL("wss://backboard.railway.app/graphql/internal"),
	)
	if err != nil {
		logger.Stderr.Error("error creating graphql client", logger.ErrAttr(err))
		return 1
	}

	allServicesExist, foundServices, missingServices, err := railway.VerifyAllServicesExistWithinEnvironment(gqlClient, config.Global.ServiceIds, config.Global.EnvironmentId)
	if err != nil {
		logger.Stderr.Error("error verifying if services exist within the environment", logger.ErrAttr(err))
		return 1
	}

	if !allServicesExist {
		logger.Stderr.Error("all services must exist within the environment set by the LOCOMOTIVE_ENVIRONMENT_ID variable",
			slog.Any("missing_service_ids", missingServices),
			slog.Any("configured_service_ids", config.Global.ServiceIds),
			slog.Any("found_service_ids", foundServices),
			slog.Any("environment_id", config.Global.EnvironmentId),
		)

		return 1
	}

	if config.Otel.Enabled {
		if err := otel.Setup(context.Background(), otel.Config{
			Enabled:         config.Otel.Enabled,
			Endpoint:        config.Otel.Endpoint,
			ServiceName:     config.Otel.ServiceName,
			EnvironmentName: config.Otel.EnvironmentName,
		}); err != nil {
			logger.Stderr.Error("failed to setup OTEL", logger.ErrAttr(err))
			return 1
		}
		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := otel.Shutdown(shutdownCtx); err != nil {
				logger.Stderr.Error("failed to shutdown OTEL", logger.ErrAttr(err))
			}
		}()
	}

	readyAttrs := []any{
		slog.Any("service_ids", config.Global.ServiceIds),
		slog.Any("environment_id", config.Global.EnvironmentId),
		slog.Bool("enable_http_logs", config.Global.EnableHttpLogs),
		slog.Bool("enable_deploy_logs", config.Global.EnableDeployLogs),
		slog.Bool("otel_enabled", config.Otel.Enabled),
	}

	if config.Otel.Enabled {
		readyAttrs = append(readyAttrs,
			slog.String("otel_endpoint", config.Otel.Endpoint),
			slog.String("otel_service_name", config.Otel.ServiceName),
		)
	} else {
		readyAttrs = append(readyAttrs,
			slog.String("webhook_url_host", config.Global.WebhookUrl.Host),
			slog.Any("webhook_mode", config.Global.WebhookMode),
		)
	}

	logger.Stdout.Info("The locomotive is ready to depart...", readyAttrs...)

	var deployLogs, httpLogs logCounts

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	reportStatusAsync(ctx, &deployLogs, &httpLogs)

	errGroup, _ := errgroup.NewErrGroup(ctx)
	defer errGroup.Cancel()

	errGroup.Go(func(ctx context.Context) error {
		if !config.Global.EnableDeployLogs {
			logger.Stdout.Info("Deploy log transport is disabled. To enable it, set LOCOMOTIVE_ENABLE_DEPLOY_LOGS=true")
			return nil
		}

		if config.Otel.Enabled {
			return pipeline.RunOtel(ctx, gqlClient, config.Global.EnvironmentId, config.Global.ServiceIds, environment_logs.SubscribeToServiceLogs, otel.EmitEnvironmentLogs, &deployLogs.processed)
		}

		deployLogsPipeline := pipeline.NewLogPipeline(
			pipeline.Client(gqlClient),
			pipeline.EnvironmentID(config.Global.EnvironmentId),
			pipeline.ServiceIDs(config.Global.ServiceIds),
			pipeline.Serialize(serializer.DeployLogs),
			pipeline.Subscribe(environment_logs.SubscribeToServiceLogs),
			pipeline.Processed(&deployLogs.processed),
			pipeline.Failed(&deployLogs.failed),
		)

		return deployLogsPipeline.Run(ctx)
	})

	errGroup.Go(func(ctx context.Context) error {
		if !config.Global.EnableHttpLogs {
			logger.Stdout.Info("HTTP log transport is disabled. To enable it, set LOCOMOTIVE_ENABLE_HTTP_LOGS=true")
			return nil
		}

		if config.Otel.Enabled {
			return pipeline.RunOtel(ctx, gqlClient, config.Global.EnvironmentId, config.Global.ServiceIds, http_logs.SubscribeToHttpLogs, otel.EmitHttpLogs, &httpLogs.processed)
		}

		httpLogsPipeline := pipeline.NewLogPipeline(
			pipeline.Client(gqlClient),
			pipeline.EnvironmentID(config.Global.EnvironmentId),
			pipeline.ServiceIDs(config.Global.ServiceIds),
			pipeline.Serialize(serializer.HttpLogs),
			pipeline.Subscribe(http_logs.SubscribeToHttpLogs),
			pipeline.Processed(&httpLogs.processed),
			pipeline.Failed(&httpLogs.failed),
		)

		return httpLogsPipeline.Run(ctx)
	})

	logger.Stdout.Info("The locomotive is waiting for cargo...")

	if err := errGroup.Wait(); err != nil {
		logger.Stderr.Error("error returned from subscription(s)", logger.ErrAttr(err))
		return 1
	}

	return 0
}
