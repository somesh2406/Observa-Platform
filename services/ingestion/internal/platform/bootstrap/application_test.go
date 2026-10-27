package bootstrap

import (
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"
)

func TestRunCleansUpWhenServerFailsToStart(t *testing.T) {
	cleaned := false
	app := &Application{
		server:          &http.Server{Addr: "bad address", Handler: http.NewServeMux()},
		logger:          slog.New(slog.NewTextHandler(io.Discard, nil)),
		shutdownTimeout: time.Second,
		cleanup: func() {
			cleaned = true
		},
	}
	err := app.Run()
	if err == nil {
		t.Fatal("expected listen error")
	}
	if !cleaned {
		t.Fatal("expected resources to be cleaned up after server startup failure")
	}
}

func TestCloseResourcesIsIdempotent(t *testing.T) {
	calls := 0
	app := &Application{cleanup: func() { calls++ }}
	app.closeResources()
	app.closeResources()
	if calls != 1 {
		t.Fatalf("cleanup calls=%d, want 1", calls)
	}
}
