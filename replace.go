package sloggcp

import "log/slog"

// Key names handled by this package.
const (
	SeverityKey       = "severity"                              // [slog.LevelKey] replacement
	MessageKey        = "message"                               // [slog.MessageKey] replacement
	SourceLocationKey = "logging.googleapis.com/sourceLocation" // [slog.SourceKey] replacement
	TimKey            = slog.TimeKey                            // time key (no replacement needed)
)

// Severity values used by GCP logging.
const (
	DebugSeverity   = "DEBUG"
	InfoSeverity    = "INFO"
	WarningSeverity = "WARNING"
	ErrorSeverity   = "ERROR"
	DefaultSeverity = "DEFAULT"
)

// ReplaceAttr replaces slog default attributes with GCP compatible ones
// https://cloud.google.com/logging/docs/structured-logging
// https://cloud.google.com/logging/docs/agent/logging/configuration#special-fields
func ReplaceAttr(groups []string, a slog.Attr) slog.Attr {
	// only handle top-level attributes
	if len(groups) > 0 {
		return a
	}
	switch a.Key {
	case slog.LevelKey:
		return replaceLevelAttr(a)
	case slog.SourceKey:
		a.Key = SourceLocationKey
	case slog.MessageKey:
		a.Key = MessageKey
	case slog.TimeKey:
		// no replacement needed
	}
	return a
}

var (
	severityDebug   = slog.String(SeverityKey, DebugSeverity)
	severityInfo    = slog.String(SeverityKey, InfoSeverity)
	severityWarn    = slog.String(SeverityKey, WarningSeverity)
	severityError   = slog.String(SeverityKey, ErrorSeverity)
	severityDefault = slog.String(SeverityKey, DefaultSeverity)
)

func replaceLevelAttr(a slog.Attr) slog.Attr {
	logLevel, ok := a.Value.Any().(slog.Level)
	if !ok {
		return severityDefault
	}
	switch logLevel {
	case slog.LevelDebug:
		return severityDebug
	case slog.LevelInfo:
		return severityInfo
	case slog.LevelWarn:
		return severityWarn
	case slog.LevelError:
		return severityError
	default:
		return severityDefault
	}
}
