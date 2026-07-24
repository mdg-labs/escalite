package log

import (
	"log/slog"
	"os"
	"strings"
)

// Context carries optional structured log fields from request or job scope.
type Context struct {
	OrgID     string
	ServiceID string
}

// NewJSONLogger returns a JSON slog logger for the given service and level.
func NewJSONLogger(serviceName, level string) *slog.Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: parseLevel(level),
	})
	return slog.New(handler).With("service", serviceName)
}

// WithContext attaches org_id and service_id when present.
func WithContext(logger *slog.Logger, ctx Context) *slog.Logger {
	attrs := make([]any, 0, 4)
	if ctx.OrgID != "" {
		attrs = append(attrs, "org_id", ctx.OrgID)
	}
	if ctx.ServiceID != "" {
		attrs = append(attrs, "service_id", ctx.ServiceID)
	}
	if len(attrs) == 0 {
		return logger
	}
	return logger.With(attrs...)
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
