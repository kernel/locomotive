package pipeline

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/brody192/locomotive/internal/logger"
	"github.com/brody192/locomotive/internal/queue"
	"github.com/brody192/locomotive/internal/railway"
	"github.com/brody192/locomotive/internal/railway/subscribe/environment_logs"
	"github.com/brody192/locomotive/internal/railway/subscribe/http_logs"
	"github.com/flexstack/uuid"
)

// RunOtel subscribes to a stream of logs of type T and emits them straight to the OTLP
// exporter, bypassing the serialize/webhook path used by LogPipeline. The OTLP SDK owns
// batching and delivery to the collector, so there is no dispatcher/queue here — the
// subscription itself is still wrapped in the same retry/backoff as LogPipeline.Run.
func RunOtel[T Log](
	ctx context.Context,
	client *railway.GraphQLClient,
	environmentId uuid.UUID,
	serviceIds []uuid.UUID,
	subscribe func(ctx context.Context, g *railway.GraphQLClient, track chan<- []T, environmentId uuid.UUID, serviceIds []uuid.UUID) error,
	emit func(ctx context.Context, logs []T) error,
	processed *atomic.Int64,
) error {
	var zero T
	var name string
	switch any(zero).(type) {
	case environment_logs.EnvironmentLogWithMetadata:
		name = deployLogsName
	case http_logs.DeploymentHttpLogWithMetadata:
		name = httpLogsName
	default:
		return fmt.Errorf("unsupported pipeline log type %T", zero)
	}

	pipeCtx, pipeCancel := context.WithCancel(ctx)
	defer pipeCancel()

	track := make(chan []T, 100)

	go func() {
		for {
			select {
			case <-pipeCtx.Done():
				return
			case logs := <-track:
				if err := emit(pipeCtx, logs); err != nil {
					logger.Stderr.Error("failed to emit logs via otel", logger.ErrAttr(err))
					continue
				}

				processed.Add(int64(len(logs)))
			}
		}
	}()

	if err := queue.RetryBackoff(
		pipeCtx,
		queue.Name((name + subscriptionNameSuffix)),
		queue.MaxRetries(10),
		queue.InitialBackoff(1*time.Second),
		queue.MaxBackoff(30*time.Second),
		queue.BackoffMultiplier(2),
		queue.BackoffJitter(0.5),
		func(ctx context.Context) error {
			if err := subscribe(ctx, client, track, environmentId, serviceIds); err != nil {
				if errors.Is(err, context.Canceled) {
					return nil
				}

				return queue.Retryable(err)
			}

			return nil
		},
	); err != nil {
		return err
	}

	logger.Stdout.Debug(fmt.Sprintf("%s subscription ended", name))

	return nil
}
