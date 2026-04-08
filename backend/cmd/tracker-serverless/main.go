package main

import (
	"context"
	"expense-tracker/backend/config"
	dbstore "expense-tracker/backend/db"
	"expense-tracker/backend/internal/serverless"
	trackerapp "expense-tracker/backend/internal/tracker"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/awslabs/aws-lambda-go-api-proxy/httpadapter"
)

func closeDBPool(storage interface{ Close() error }) {
	if err := storage.Close(); err != nil {
		log.Printf("failed to close db pool: %v", err)
	}
}

func main() {
	cfg := config.Envs

	storage, err := dbstore.NewPostgreSQLStorage(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer closeDBPool(storage)

	if err := storage.Ping(); err != nil {
		log.Fatal(err)
	}

	handler := trackerapp.NewHandler(storage)
	if config.Envs.GoogleExchangeModeIs(config.GoogleExchangeUpstreamVerified) {
		handler = serverless.WrapWithGoogleAuthorizerClaims(handler)
	}

	adapter := httpadapter.NewV2(handler)

	lambda.Start(func(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
		return adapter.ProxyWithContext(ctx, req)
	})
}
