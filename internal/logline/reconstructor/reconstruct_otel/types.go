package reconstruct_otel

type logsData struct {
	ResourceLogs []resourceLog `json:"resourceLogs"`
}

type resourceLog struct {
	Resource  resource   `json:"resource"`
	ScopeLogs []scopeLog `json:"scopeLogs"`
}

type resource struct {
	Attributes []attribute `json:"attributes"`
}

type scopeLog struct {
	Scope      scope       `json:"scope"`
	LogRecords []logRecord `json:"logRecords"`
}

type scope struct {
	Name string `json:"name"`
}

type logRecord struct {
	TimeUnixNano         string      `json:"timeUnixNano"`
	ObservedTimeUnixNano string      `json:"observedTimeUnixNano"`
	SeverityNumber       int         `json:"severityNumber"`
	SeverityText         string      `json:"severityText"`
	Body                 body        `json:"body"`
	Attributes           []attribute `json:"attributes,omitempty"`
}

type body struct {
	StringValue string `json:"stringValue"`
}

type attribute struct {
	Key   string         `json:"key"`
	Value attributeValue `json:"value"`
}

type attributeValue struct {
	StringValue *string `json:"stringValue,omitempty"`
	IntValue    *string `json:"intValue,omitempty"`
}
