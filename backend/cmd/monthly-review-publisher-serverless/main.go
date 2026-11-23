package main

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"expense-tracker/backend/config"
	dbstore "expense-tracker/backend/db"
	"expense-tracker/backend/services/monthlyreview"

	"github.com/aws/aws-lambda-go/lambda"
)

type publicationRequest struct {
	Action string `json:"action"`
}

func main() {
	lambda.Start(func(ctx context.Context, request publicationRequest) (monthlyreview.PublicationResult, error) {
		return handle(ctx, request, publish)
	})
}

func handle(ctx context.Context, request publicationRequest, run func(context.Context) (monthlyreview.PublicationResult, error)) (monthlyreview.PublicationResult, error) {
	if request.Action != "publish_monthly_reviews" {
		return monthlyreview.PublicationResult{}, errors.New("unsupported monthly review publisher action")
	}

	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	result, err := run(ctx)
	if err != nil {
		return monthlyreview.PublicationResult{}, errors.New("monthly review publication failed")
	}
	slog.Info("monthly review publication complete", "published", result.Published)
	return result, nil
}

func publish(ctx context.Context) (monthlyreview.PublicationResult, error) {
	database, err := dbstore.NewPostgreSQLStorage(config.Envs)
	if err != nil {
		return monthlyreview.PublicationResult{}, err
	}
	defer database.Close()
	return monthlyreview.NewStore(database).PublishEligible(ctx, time.Now().UTC(), 25)
}
