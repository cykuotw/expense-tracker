package main

import (
	"context"
	"errors"
	"time"

	"expense-tracker/backend/config"
	dbstore "expense-tracker/backend/db"
	"expense-tracker/backend/services/notification"

	"github.com/aws/aws-lambda-go/lambda"
)

func main() { lambda.Start(handle) }

func handle(ctx context.Context, request notification.DeliveryRequest) (notification.DeliveryBatch, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	database, err := dbstore.NewPostgreSQLStorage(config.Envs)
	if err != nil {
		return notification.DeliveryBatch{}, errors.New("open delivery database failed")
	}
	defer database.Close()
	batch, err := notification.NewStore(database).HandleDeliveryRequest(ctx, request)
	if err != nil {
		// Do not expose database errors, credentials, or subscription data in Lambda logs.
		return notification.DeliveryBatch{}, errors.New("delivery database operation failed")
	}
	return batch, nil
}
