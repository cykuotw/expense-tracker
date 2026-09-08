package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"expense-tracker/backend/services/notification"

	"github.com/aws/aws-lambda-go/lambda"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	lambdaclient "github.com/aws/aws-sdk-go-v2/service/lambda"
)

func main() {
	lambda.Start(handle)
}

func handle(ctx context.Context) error {
	function := strings.TrimSpace(os.Getenv("PUSH_DELIVERY_FUNCTION_NAME"))
	if function == "" {
		return errors.New("PUSH_DELIVERY_FUNCTION_NAME is required")
	}
	configuration, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRetryMaxAttempts(1))
	if err != nil {
		return errors.New("load AWS configuration failed")
	}
	store := notification.NewLambdaStore(lambdaclient.NewFromConfig(configuration), function)

	sender, err := notification.NewSender(
		store,
		strings.TrimSpace(os.Getenv("WEB_PUSH_VAPID_PUBLIC_KEY")),
		strings.TrimSpace(os.Getenv("WEB_PUSH_VAPID_PRIVATE_KEY")),
		strings.TrimSpace(os.Getenv("WEB_PUSH_VAPID_SUBJECT")),
	)
	if err != nil {
		return err
	}
	if err := sender.RunOnce(ctx); err != nil {
		return fmt.Errorf("send push notifications: %w", err)
	}
	return nil
}
