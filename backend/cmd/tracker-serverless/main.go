package main

import (
	"context"
	"expense-tracker/backend/config"
	dbstore "expense-tracker/backend/db"
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

	adapter := httpadapter.New(trackerapp.NewHandler(storage))

	lambda.Start(func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		return adapter.ProxyWithContext(ctx, req)
	})
}
