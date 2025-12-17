package sloggcp

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"reflect"
	"testing"
)

type expectSchema struct {
	Type           string         `json:"@type"`
	Message        string         `json:"message"`
	Severity       string         `json:"severity"`
	Source         testSource     `json:"logging.googleapis.com/sourceLocation"`
	Error          any            `json:"error"`
	Foo            fooType        `json:"foo"`
	ReportLocation ReportLocation `json:"reportLocation"`
}

type fooType struct {
	Bar   string `json:"bar"`
	Baz   int    `json:"baz"`
	Error string `json:"error"`
}

func (f fooType) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("bar", f.Bar),
		slog.Int("baz", f.Baz),
	)
}

var fooTypeTest = fooType{
	Bar: "baz",
	Baz: 42,
}

type testSource struct {
	Function string `json:"function"`
	// File and line are omitted.
}

func TestHandler(t *testing.T) {
	var buf bytes.Buffer
	dec := json.NewDecoder(&buf)
	tests := []struct {
		name string
		opts *slog.HandlerOptions
		log  func(logger *slog.Logger)
		want *expectSchema
	}{
		{
			name: "debug disabled",
			opts: nil,
			log: func(logger *slog.Logger) {
				logger.Debug("this is debug", "foo", fooTypeTest)
			},
			want: nil,
		},
		{
			name: "log info message",
			opts: nil,
			log: func(logger *slog.Logger) {
				logger.Info("this is info", "foo", fooTypeTest)
			},
			want: &expectSchema{
				Message:  "this is info",
				Severity: InfoSeverity,
				Foo:      fooTypeTest,
			},
		},
		{
			name: "log info message, with source",
			opts: &slog.HandlerOptions{
				AddSource: true,
			},
			log: func(logger *slog.Logger) {
				logger.Info("this is info", "foo", fooTypeTest)
			},
			want: &expectSchema{
				Message:  "this is info",
				Severity: InfoSeverity,
				Source: testSource{
					Function: "github.com/muhlemmer/sloggcp.TestHandler.func3",
				},
				Foo: fooTypeTest,
			},
		},
		{
			name: "log warn with group and attrs",
			log: func(logger *slog.Logger) {
				logger = logger.WithGroup("foo")
				logger = logger.With(
					slog.String("bar", "baz"),
					slog.Int("baz", 42),
				)
				logger.Warn("warn message", slog.String("error", "grouped error"))
			},
			want: &expectSchema{
				Message:  "warn message",
				Severity: WarningSeverity,
				Foo: fooType{
					Bar:   "baz",
					Baz:   42,
					Error: "grouped error",
				},
			},
		},
		{
			name: "log info grouped without attrs",
			log: func(logger *slog.Logger) {
				logger = logger.WithGroup("foo")
				logger.Info("info message")
			},
			want: &expectSchema{
				Message:  "info message",
				Severity: InfoSeverity,
			},
		},
		{
			name: "log error string",
			log: func(logger *slog.Logger) {
				logger.Error("error message", "foo", fooTypeTest, slog.String("error", "something went wrong"))
			},
			want: &expectSchema{
				Type:     ErrorReportTypeValue,
				Message:  "something went wrong",
				Severity: ErrorSeverity,
				Error:    "something went wrong",
				Foo:      fooTypeTest,
			},
		},
		{
			name: "log error string from WithAttrs",
			log: func(logger *slog.Logger) {
				logger = logger.With(
					slog.String("error", "something went wrong"),
					slog.Any("foo", fooTypeTest),
				)
				logger.Error("error message")
			},
			want: &expectSchema{
				Type:     ErrorReportTypeValue,
				Message:  "something went wrong",
				Severity: ErrorSeverity,
				Error:    "something went wrong",
				Foo:      fooTypeTest,
			},
		},
		{
			name: "log error string after ReplaceAttr",
			opts: &slog.HandlerOptions{
				ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
					if len(groups) != 0 {
						return a
					}
					if a.Key == "err" {
						a.Key = ErrorKey
					}
					return a
				},
			},
			log: func(logger *slog.Logger) {
				logger = logger.With(
					slog.String("error", "something went wrong"),
				)
				logger.Error("error message", "foo", fooTypeTest)
			},
			want: &expectSchema{
				Type:     ErrorReportTypeValue,
				Message:  "something went wrong",
				Severity: ErrorSeverity,
				Error:    "something went wrong",
				Foo:      fooTypeTest,
			},
		},
		{
			name: "log standard error",
			log: func(logger *slog.Logger) {
				logger.Error("error message", "error", errors.New("something went wrong"))
			},
			want: &expectSchema{
				Type:     ErrorReportTypeValue,
				Message:  "something went wrong",
				Severity: ErrorSeverity,
				Error:    "something went wrong",
			},
		},
		{
			name: "log ReportLocationError",
			log: func(logger *slog.Logger) {
				logger.Error("error message", "error", mockReportLocationError{})
			},
			want: &expectSchema{
				Type:           ErrorReportTypeValue,
				Message:        "mockReportLocationError",
				Severity:       ErrorSeverity,
				Error:          "mockReportLocationError",
				ReportLocation: mockReportLocation,
			},
		},
		{
			name: "log StackTraceError",
			log: func(logger *slog.Logger) {
				logger.Error("error message", "error", mockStackTraceError{})
			},
			want: &expectSchema{
				Type:     ErrorReportTypeValue,
				Message:  "stack",
				Severity: ErrorSeverity,
				Error:    "mockStackTraceError",
			},
		},
		{
			name: "log stackAndReport",
			log: func(logger *slog.Logger) {
				logger.Error("error message", "error", mockStackAndReport{})
			},
			want: &expectSchema{
				Type:           ErrorReportTypeValue,
				Message:        "stack",
				Severity:       ErrorSeverity,
				Error:          "mockStackAndReport",
				ReportLocation: mockReportLocation,
			},
		},
		{
			name: "log stackAndReportValuer",
			log: func(logger *slog.Logger) {
				logger.Error("error message", "error", mockStackAndReportValuer{})
			},
			want: &expectSchema{
				Type:     ErrorReportTypeValue,
				Message:  "stack",
				Severity: ErrorSeverity,
				Error: map[string]any{
					"key1": "value1",
					"key2": float64(42),
				},
				ReportLocation: mockReportLocation,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer buf.Reset()

			h := NewErrorReportingHandler(&buf, tt.opts)
			logger := slog.New(h)
			tt.log(logger)
			if tt.want == nil {
				if buf.Len() != 0 {
					t.Errorf("log wrote data, but want is nil: %q", buf.String())
				}
				return
			}

			var got expectSchema
			if err := dec.Decode(&got); err != nil {
				t.Fatalf("Failed to decode log output: %v", err)
			}
			if !reflect.DeepEqual(&got, tt.want) {
				t.Errorf("log output = %+v, want %+v", &got, tt.want)
			}
		})
	}
}

func Test_severityFromLevel(t *testing.T) {
	tests := []struct {
		name  string
		level slog.Level
		want  string
	}{
		{
			name:  "Debug",
			level: slog.LevelDebug,
			want:  DebugSeverity,
		},
		{
			name:  "Info",
			level: slog.LevelInfo,
			want:  InfoSeverity,
		},
		{
			name:  "Warn",
			level: slog.LevelWarn,
			want:  WarningSeverity,
		},
		{
			name:  "Error",
			level: slog.LevelError,
			want:  ErrorSeverity,
		},
		{
			name:  "Default",
			level: slog.Level(-1),
			want:  DefaultSeverity,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := severityFromLevel(tt.level)
			if got != tt.want {
				t.Errorf("severityFromLevel() = %v, want %v", got, tt.want)
			}
		})
	}
}
