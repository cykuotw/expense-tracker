package ocrusage

import (
	"context"
	"encoding/json/v2"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
)

type Invoker interface {
	Invoke(context.Context, *lambda.InvokeInput, ...func(*lambda.Options)) (*lambda.InvokeOutput, error)
}

// Enqueue waits only for AWS acceptance. Processing and retries happen off the
// scan path; callers must log a failure without retrying the paid Gemini call.
func Enqueue(ctx context.Context, client Invoker, target string, r Record) error {
	if target == "" {
		return errors.New("OCR usage target is not configured")
	}
	if err := r.Validate(); err != nil {
		return err
	}
	payload, err := json.Marshal(Event{Type: EventType, Usage: r})
	if err != nil {
		return err
	}
	response, err := client.Invoke(ctx, &lambda.InvokeInput{
		FunctionName: aws.String(target), InvocationType: types.InvocationTypeEvent, Payload: payload,
	})
	if err != nil {
		return err
	}
	if response == nil || response.StatusCode != 202 {
		return errors.New("OCR usage event was not accepted")
	}
	return nil
}
