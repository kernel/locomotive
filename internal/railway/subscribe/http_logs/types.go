package http_logs

import (
	"encoding/json"
	"time"
)

type DeploymentHttpLogMetadata map[string]string

type DeploymentHttpLogWithMetadata struct {
	Timestamp time.Time

	Log        json.RawMessage
	Path       string
	StatusCode int64

	Metadata DeploymentHttpLogMetadata
}

func (d DeploymentHttpLogWithMetadata) GetMetadata() map[string]string {
	return d.Metadata
}
