package main

import (
	"flag"
	"log/slog"
	"os"

	"github.com/observa-platform/observa/services/ingestion/internal/platform/bootstrap"
)

func main() {
	configPath := flag.String("config", "", "Path to configuration file")
	flag.Parse()

	// Fallback to environment variable if flag is missing
	if *configPath == "" {
		*configPath = os.Getenv("OBSERVA_CONFIG_FILE")
	}

	app, err := bootstrap.Build(*configPath)
	if err != nil {
		slog.Error("application build failed", "error", err)
		os.Exit(1)
	}

	if err := app.Run(); err != nil {
		slog.Error("application runtime error", "error", err)
		os.Exit(1)
	}
}
