package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mdg-labs/escalite/services/integrations/internal/config"
	"github.com/mdg-labs/escalite/services/integrations/internal/log"
	"github.com/mdg-labs/escalite/services/integrations/internal/server"
)

const serviceName = "integrations"

func main() {
	os.Exit(run())
}

func run() int {
	cfg, err := config.Load(config.Options{
		ServiceName:       serviceName,
		DefaultListenAddr: ":8082",
		RequireDatabase:   false,
	})
	if err != nil {
		slog.Error("configuration error", "error", err.Error())
		return 1
	}

	logger := log.NewJSONLogger(serviceName, cfg.LogLevel)
	logger.Info("starting service", "listen_addr", cfg.ListenAddr)

	handler := server.New(logger)
	httpServer := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		logger.Error("server error", "error", err)
		return 1
	case sig := <-stop:
		logger.Info("shutdown signal received", "signal", sig.String())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		return 1
	}

	logger.Info("service stopped")
	return 0
}
