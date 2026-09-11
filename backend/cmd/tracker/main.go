package main

import (
	"context"
	"expense-tracker/backend/config"
	dbstore "expense-tracker/backend/db"
	"expense-tracker/backend/internal/observability"
	"expense-tracker/backend/internal/requestserver"
	trackerapp "expense-tracker/backend/internal/tracker"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func closeDBPool(logger *slog.Logger, storage interface{ Close() error }) {
	if err := storage.Close(); err != nil {
		logger.Error("database_pool_close_failed", slog.String("error_type", fmt.Sprintf("%T", err)))
	}
}

func exitWithError(logger *slog.Logger, event string, err error) {
	logger.Error(event, slog.String("error_type", fmt.Sprintf("%T", err)))
	os.Exit(1)
}

func main() {
	logger := observability.NewLogger("release", os.Stderr)
	cfg, err := config.Load()
	if err != nil {
		exitWithError(logger, "configuration_load_failed", err)
	}
	logger = observability.NewLogger(cfg.Mode, os.Stdout)

	storage, err := requestserver.OpenDatabase(cfg, config.RequestServerStandalone, dbstore.NewPostgreSQLStorage)
	if err != nil {
		exitWithError(logger, "database_open_failed", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	handler := trackerapp.NewHandler(storage, trackerapp.WithLogger(logger))
	apiServer := trackerapp.NewHTTPServer(cfg.BackendURL, handler)
	go func() {
		logger.Info("server_started")
		if err := apiServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			exitWithError(logger, "server_failed", err)
		}
	}()

	<-ctx.Done()

	stop()
	logger.Info("server_shutdown_started")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := apiServer.Shutdown(shutdownCtx); err != nil {
		closeDBPool(logger, storage)
		exitWithError(logger, "server_shutdown_failed", err)
	}
	closeDBPool(logger, storage)
	logger.Info("server_stopped")
}
