package main

import (
	"context"
	"os"
	"strings"
	"sync"

	"expense-tracker/backend/internal/errornotifier"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

var handlerCache struct {
	sync.Mutex
	handler *errornotifier.Handler
}

func main() {
	lambda.Start(handle)
}

func handle(ctx context.Context, event events.CloudwatchLogsEvent) error {
	handler, err := runtimeHandler()
	if err != nil {
		return err
	}
	return handler.Handle(ctx, event)
}

func runtimeHandler() (*errornotifier.Handler, error) {
	handlerCache.Lock()
	defer handlerCache.Unlock()
	if handlerCache.handler != nil {
		return handlerCache.handler, nil
	}

	provider, err := errornotifier.NewWebhookProvider(
		strings.TrimSpace(os.Getenv("DISCORD_WEBHOOK_URL")),
	)
	if err != nil {
		return nil, err
	}
	discord, err := errornotifier.NewDiscordClient(errornotifier.NewHTTPClient())
	if err != nil {
		return nil, err
	}
	handler, err := errornotifier.NewHandler(
		strings.TrimSpace(os.Getenv("DEPLOYMENT_ENVIRONMENT")),
		strings.TrimSpace(os.Getenv("AWS_LAMBDA_FUNCTION_NAME")),
		provider,
		discord,
	)
	if err != nil {
		return nil, err
	}
	handlerCache.handler = handler
	return handler, nil
}
