package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"expense-tracker/backend/services/notification"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	lambdaclient "github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/s3"
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
	if bucket := strings.TrimSpace(os.Getenv("OPERATING_HOURS_BUCKET")); bucket != "" {
		policyCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		allowed, err := notification.CheckSenderHours(policyCtx, func(ctx context.Context) (io.ReadCloser, error) {
			result, err := s3.NewFromConfig(configuration).GetObject(ctx, &s3.GetObjectInput{
				Bucket: aws.String(bucket), Key: aws.String("operating-hours.json"),
			})
			if err != nil {
				return nil, err
			}
			return result.Body, nil
		}, time.Now())
		if err != nil {
			return err
		}
		if !allowed {
			return nil
		}
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
