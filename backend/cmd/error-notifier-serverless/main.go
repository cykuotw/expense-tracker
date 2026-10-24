package main

import (
	"context"
	"encoding/json"
	jsonv2 "encoding/json/v2"
	"errors"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
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

func handle(ctx context.Context, raw json.RawMessage) error {
	var envelope struct {
		Records json.RawMessage `json:"Records"`
	}
	if jsonv2.Unmarshal(raw, &envelope) != nil {
		return errors.New("invalid notifier event")
	}
	if len(envelope.Records) > 0 {
		var event events.S3Event
		if jsonv2.Unmarshal(raw, &event) != nil {
			return errors.New("invalid database notifier event")
		}
		provider, err := errornotifier.NewWebhookProvider(strings.TrimSpace(os.Getenv("DATABASE_DISCORD_WEBHOOK_URL")))
		if err != nil {
			return err
		}
		client, err := errornotifier.NewDiscordClient(errornotifier.NewHTTPClient())
		if err != nil {
			return err
		}
		cfg, err := awsconfig.LoadDefaultConfig(ctx)
		if err != nil {
			return errors.New("database notifier configuration unavailable")
		}
		handler, err := errornotifier.NewDatabaseHandler(os.Getenv("DATABASE_ALERT_BUCKET"), os.Getenv("DEPLOYMENT_ENVIRONMENT"), s3.NewFromConfig(cfg), provider, client)
		if err != nil {
			return err
		}
		return handler.Handle(ctx, event)
	}
	var event events.CloudwatchLogsEvent
	if jsonv2.Unmarshal(raw, &event) != nil {
		return errors.New("invalid application notifier event")
	}
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
