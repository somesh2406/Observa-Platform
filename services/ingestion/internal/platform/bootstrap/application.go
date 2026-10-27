package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type Application struct {
	server          *http.Server
	logger          *slog.Logger
	shutdownTimeout time.Duration
	cleanup         func()
	cleanupOnce     sync.Once
}

func (a *Application) closeResources() {
	a.cleanupOnce.Do(func() {
		if a.cleanup != nil {
			a.cleanup()
		}
	})
}

func (a *Application) Run() error {
	errCh := make(chan error, 1)
	go func() {
		a.logger.Info("ingestion service starting", "address", a.server.Addr)
		errCh <- a.server.ListenAndServe()
	}()

	sigCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case <-sigCtx.Done():
		a.logger.Info("shutdown signal received; draining HTTP requests")
		ctx, cancel := context.WithTimeout(context.Background(), a.shutdownTimeout)
		defer cancel()

		// Stop accepting new work and allow active requests to finish before
		// closing Kafka. Closing the producer first can fail in-flight publishes.
		shutdownErr := a.server.Shutdown(ctx)
		if shutdownErr != nil {
			// Force-close connections if graceful draining exceeded its deadline.
			_ = a.server.Close()
		}
		a.closeResources()
		if shutdownErr != nil {
			return shutdownErr
		}
		return nil

	case err := <-errCh:
		a.closeResources()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	}
}
