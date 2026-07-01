package config

import (
	"net/url"
	"time"

	"github.com/brody192/locomotive/internal/railway/subscribe/environment_logs"
	"github.com/brody192/locomotive/internal/railway/subscribe/http_logs"
	"github.com/flexstack/uuid"
)

type (
	AdditionalHeaders map[string]string

	WebhookMode string
)

type WebhookConfig struct {
	ExpectedHostContains []string
	ExpectedHeaders      []string

	// DefaultHeaders are headers this mode forces on every request (e.g. Content-Type).
	// Distinct from the user-supplied AdditionalHeaders.
	DefaultHeaders map[string]string

	EnvironmentLogReconstructorFunc func([]environment_logs.EnvironmentLogWithMetadata) ([]byte, error)
	HTTPLogReconstructorFunc        func([]http_logs.DeploymentHttpLogWithMetadata) ([]byte, error)
}

type config struct {
	RailwayApiKey uuid.UUID   `env:"RAILWAY_API_KEY,required,notEmpty"`
	EnvironmentId uuid.UUID   `env:"ENVIRONMENT_ID,required,notEmpty"`
	ServiceIds    []uuid.UUID `env:"SERVICE_IDS,required,notEmpty"`

	WebhookUrl        url.URL           `env:"WEBHOOK_URL"`
	AdditionalHeaders AdditionalHeaders `env:"ADDITIONAL_HEADERS"`
	WebhookMode       WebhookMode       `env:"WEBHOOK_MODE" envDefault:"json"`

	ReportStatusEvery time.Duration `env:"REPORT_STATUS_EVERY" envDefault:"1m"`

	EnableHttpLogs   bool `env:"ENABLE_HTTP_LOGS" envDefault:"false"`
	EnableDeployLogs bool `env:"ENABLE_DEPLOY_LOGS" envDefault:"true"`
}

// OtelConfig selects OTLP gRPC export instead of the webhook path. Its env vars use the
// conventional OTEL_ names (no LOCOMOTIVE_ prefix). When Enabled, logs are emitted to the
// OTLP endpoint under ServiceName rather than serialized and POSTed to WebhookUrl.
type OtelConfig struct {
	Enabled         bool   `env:"OTEL_ENABLED" envDefault:"false"`
	Endpoint        string `env:"OTEL_EXPORTER_OTLP_ENDPOINT"`
	ServiceName     string `env:"OTEL_SERVICE_NAME"`
	EnvironmentName string `env:"OTEL_ENVIRONMENT_NAME"`
}
