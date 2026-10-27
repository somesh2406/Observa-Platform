package logger

import (
	"log/slog"
	"os"
	"strings"

	"github.com/observa-platform/observa/services/ingestion/internal/config"
)

func New(cfg config.LoggerConfig, serviceName string) *slog.Logger {
	var level slog.Level

	switch strings.ToUpper(strings.TrimSpace(cfg.Level)) {
	case "ERROR":
		level = slog.LevelError
	case "WARN":
		level = slog.LevelWarn
	case "DEBUG":
		level = slog.LevelDebug
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		AddSource: cfg.AddSource,
		Level:     level,
	}

	var handler slog.Handler
	if strings.EqualFold(cfg.Format, "text") {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	if serviceName == "" {
		serviceName = "Observa Service"
	}

	return slog.New(handler).With("service", serviceName)
}
