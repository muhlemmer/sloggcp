package sloggcp

import (
	"fmt"
	"log/slog"
	"runtime"

	_ "runtime/debug"
)

// Key by which errors are retrieved from slog attributes.
// The corresponding values can be of type [string], [error], [StackTraceError] and/or [ReportLocationError].
const (
	ErrorKey = "error"
)

// Constants for GCP error reporting attributes.
// See https://cloud.google.com/error-reporting/docs/formatting-error-messages.
const (
	ErrorReportTypeKey   = "@type"
	ErrorReportTypeValue = "type.googleapis.com/google.devtools.clouderrorreporting.v1beta1.ReportedErrorEvent"
	ReportLocationKey    = "reportLocation"
	FilePathKey          = "filePath"
	LineNumberKey        = "lineNumber"
	FunctionNameKey      = "functionName"
)

// StackTraceError is an error that provides a stack trace,
// from the point where the error was created.
// The returned stack trace must be the value returned by [debug.Stack].
type StackTraceError interface {
	error
	StackTrace() []byte
}

// ReportLocationError is an error that provides report location information,
// from the point where the error was created.
type ReportLocationError interface {
	error
	ReportLocation() *ReportLocation
}

// Make type switching easier
type stackAndReport interface {
	StackTraceError
	ReportLocationError
}

func assertErrorValue(value any) (errMsg string, reportLocation *ReportLocation) {
	switch v := value.(type) {
	case stackAndReport:
		errMsg = string(v.StackTrace())
		reportLocation = v.ReportLocation()
	case StackTraceError:
		errMsg = string(v.StackTrace())
	case ReportLocationError:
		errMsg = v.Error()
		reportLocation = v.ReportLocation()
	case error:
		errMsg = v.Error()
	case string:
		errMsg = v
	default:
		errMsg = fmt.Sprintf("!!! can't handle error report for type %T !!!", v)
		reportLocation = NewReportLocation(0)
	}
	return errMsg, reportLocation
}

type ReportLocation struct {
	FilePath     string `json:"filePath"`
	LineNumber   int    `json:"lineNumber"`
	FunctionName string `json:"functionName"`
}

// NewReportLocation based on the current call stack.
// The returned [ReportLocation] can be stored and returned
// in a [ReportLocationError].
// The skip parameter is the number of stack frames to skip
// (0 identifies the caller of NewReportLocation).
func NewReportLocation(skip int) *ReportLocation {
	pc, file, line, ok := runtime.Caller(skip + 1)
	fn := runtime.FuncForPC(pc)
	if !ok || fn == nil {
		return nil
	}
	return &ReportLocation{
		FilePath:     file,
		LineNumber:   line,
		FunctionName: fn.Name(),
	}
}

func checkAndSetErrorReport(a slog.Attr, out map[string]any) bool {
	if a.Key != ErrorKey {
		return false
	}
	value := a.Value.Any()
	errMsg, reportLocation := assertErrorValue(value)
	out[ErrorReportTypeKey] = ErrorReportTypeValue
	out[MessageKey] = errMsg
	out[ErrorKey] = value
	if reportLocation != nil {
		out[ReportLocationKey] = reportLocation
	}
	switch v := value.(type) {
	case slog.LogValuer:
		out[ErrorKey] = extractValue(v.LogValue())
	case error:
		out[ErrorKey] = v.Error()
	}

	return true
}
