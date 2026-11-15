package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"expense-tracker/backend/config"
	dbstore "expense-tracker/backend/db"
	"expense-tracker/backend/internal/dbavailability"
	"expense-tracker/backend/internal/observability"
	"expense-tracker/backend/internal/requestserver"
	"expense-tracker/backend/internal/serverless"
	trackerapp "expense-tracker/backend/internal/tracker"
	"expense-tracker/backend/services/ocrusage"
	"fmt"
	"log/slog"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/awslabs/aws-lambda-go-api-proxy/httpadapter"
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

	policy, err := dbavailability.ParsePolicy(cfg.DBOperatingHours)
	if err != nil {
		exitWithError(logger, "operating_hours_invalid", err)
	}
	availability := dbavailability.New(policy)
	storage, err := requestserver.OpenDatabase(cfg, config.RequestServerServerless, func(cfg config.Config) (*sql.DB, error) {
		return dbstore.NewRequestStorage(cfg, availability)
	})
	if err != nil {
		exitWithError(logger, "database_open_failed", err)
	}
	defer closeDBPool(logger, storage)

	handler := trackerapp.NewHandler(storage, trackerapp.WithLogger(logger), trackerapp.WithDatabaseAvailability(availability))
	if cfg.GoogleExchangeModeIs(config.GoogleExchangeUpstreamVerified) {
		handler = serverless.WrapWithGoogleAuthorizerClaims(handler)
	}
	handler = serverless.WrapWithRequestMetadata(handler)

	adapter := httpadapter.NewV2(handler)
	logger.Info("serverless_worker_started")

	usageStore := ocrusage.NewStore(storage)
	lambda.Start(func(ctx context.Context, raw json.RawMessage) (any, error) {
		return serverless.HandleWorkerEvent(ctx, raw, usageStore, adapter.ProxyWithContext)
	})
}
