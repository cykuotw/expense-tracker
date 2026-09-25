package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"expense-tracker/backend/internal/observability"
	"expense-tracker/backend/services/ocr"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-lambda-go/lambdacontext"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

func main() {
	logger := observability.NewLogger("release", os.Stdout)
	runtimeConfig, err := ocr.LoadRuntimeConfig(os.LookupEnv)
	if err != nil {
		logger.Error("ocr_configuration_load_failed", slog.String("error_type", fmt.Sprintf("%T", err)))
		os.Exit(1)
	}
	awsConfig, err := awsconfig.LoadDefaultConfig(context.Background())
	if err != nil {
		logger.Error("ocr_aws_configuration_load_failed", slog.String("error_type", fmt.Sprintf("%T", err)))
		os.Exit(1)
	}
	handler := ocr.NewStubHandler(
		runtimeConfig,
		ocr.NewDynamoReplayStore(dynamodb.NewFromConfig(awsConfig), runtimeConfig.ReplayTable),
	)
	logger.Info("ocr_serverless_started")

	lambda.Start(func(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
		startedAt := time.Now()
		response, err := handler.Handle(ctx, request)
		attributes := []any{
			slog.String("route", "POST /api/v0/ocr/drafts"),
			slog.Int("status", response.StatusCode),
			slog.Int64("latency_ms", time.Since(startedAt).Milliseconds()),
		}
		if request.RequestContext.RequestID != "" {
			attributes = append(attributes, slog.String("api_gateway_request_id", request.RequestContext.RequestID))
		}
		if lambdaContext, ok := lambdacontext.FromContext(ctx); ok {
			attributes = append(attributes, slog.String("aws_request_id", lambdaContext.AwsRequestID))
		}
		if response.StatusCode >= 500 {
			attributes = append(attributes, slog.Bool("alertable", true))
			logger.Error("ocr_request_failed", attributes...)
		} else {
			logger.Info("ocr_request_completed", attributes...)
		}
		return response, err
	})
}
