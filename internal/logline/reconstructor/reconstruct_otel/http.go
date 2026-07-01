package reconstruct_otel

import (
	"encoding/json"
	"strconv"

	"github.com/brody192/locomotive/internal/railway/subscribe/http_logs"
)

func HttpLogsOtel(logs []http_logs.DeploymentHttpLogWithMetadata) ([]byte, error) {
	data := buildLogsData(logs, func(log http_logs.DeploymentHttpLogWithMetadata, nowNano string) logRecord {
		// Derive log severity from HTTP status code
		severityText := "INFO"
		severityNumberKey := "info"
		if log.StatusCode >= 500 {
			severityText = "ERROR"
			severityNumberKey = "error"
		} else if log.StatusCode >= 400 {
			severityText = "WARN"
			severityNumberKey = "warn"
		}

		// Build attributes: status code + flattened JSON log
		logAttrs := []attribute{intAttribute("http.response.status_code", log.StatusCode)}
		logAttrs = append(logAttrs, jsonBytesToAttributes(log.Log)...)

		return logRecord{
			TimeUnixNano:         strconv.FormatInt(log.Timestamp.UnixNano(), 10),
			ObservedTimeUnixNano: nowNano,
			SeverityNumber:       getSeverityNumber(severityNumberKey),
			SeverityText:         severityText,
			Body:                 body{StringValue: log.Path},
			Attributes:           logAttrs,
		}
	})

	return json.Marshal(data)
}
