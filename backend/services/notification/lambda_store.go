package notification

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	lambdatypes "github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/google/uuid"
)

type LambdaInvoker interface {
	Invoke(context.Context, *lambda.InvokeInput, ...func(*lambda.Options)) (*lambda.InvokeOutput, error)
}

// LambdaStore contains transport only; Sender owns delivery and retry decisions.
type LambdaStore struct {
	client   LambdaInvoker
	function string
}

func NewLambdaStore(client LambdaInvoker, function string) *LambdaStore {
	return &LambdaStore{client: client, function: function}
}

func (s *LambdaStore) invoke(ctx context.Context, request DeliveryRequest) (DeliveryBatch, error) {
	payload, err := json.Marshal(request)
	if err != nil {
		return DeliveryBatch{}, err
	}
	response, err := s.client.Invoke(ctx, &lambda.InvokeInput{
		FunctionName: aws.String(s.function), InvocationType: lambdatypes.InvocationTypeRequestResponse, Payload: payload,
	})
	if err != nil {
		return DeliveryBatch{}, errors.New("delivery database invocation failed")
	}
	if response == nil || response.StatusCode != 200 || response.FunctionError != nil {
		return DeliveryBatch{}, errors.New("delivery database operation failed")
	}
	var batch DeliveryBatch
	if err := json.Unmarshal(response.Payload, &batch); err != nil {
		return DeliveryBatch{}, errors.New("invalid delivery database response")
	}
	return batch, nil
}

func (s *LambdaStore) ClaimDeliveries(ctx context.Context) (DeliveryBatch, error) {
	batch, err := s.invoke(ctx, DeliveryRequest{Action: "claim"})
	if err != nil {
		return DeliveryBatch{}, err
	}
	if batch.Token == uuid.Nil || len(batch.Deliveries) > senderBatchSize {
		return DeliveryBatch{}, errors.New("invalid delivery batch")
	}
	return batch, nil
}

func (s *LambdaStore) AcknowledgeDeliveries(ctx context.Context, token uuid.UUID, results []DeliveryResult) error {
	_, err := s.invoke(ctx, DeliveryRequest{Action: "ack", Token: token, Results: results})
	return err
}
