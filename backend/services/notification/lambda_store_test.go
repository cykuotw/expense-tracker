package notification

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	lambdatypes "github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type invokeFunc func(context.Context, *lambda.InvokeInput, ...func(*lambda.Options)) (*lambda.InvokeOutput, error)

func (f invokeFunc) Invoke(ctx context.Context, input *lambda.InvokeInput, options ...func(*lambda.Options)) (*lambda.InvokeOutput, error) {
	return f(ctx, input, options...)
}

func TestLambdaStoreClaimAndAcknowledge(t *testing.T) {
	token := uuid.New()
	var requests []DeliveryRequest
	store := NewLambdaStore(invokeFunc(func(_ context.Context, input *lambda.InvokeInput, _ ...func(*lambda.Options)) (*lambda.InvokeOutput, error) {
		require.Equal(t, "delivery", aws.ToString(input.FunctionName))
		require.Equal(t, lambdatypes.InvocationTypeRequestResponse, input.InvocationType)
		var request DeliveryRequest
		require.NoError(t, json.Unmarshal(input.Payload, &request))
		requests = append(requests, request)
		payload, err := json.Marshal(DeliveryBatch{Token: token})
		require.NoError(t, err)
		return &lambda.InvokeOutput{StatusCode: 200, Payload: payload}, nil
	}), "delivery")
	batch, err := store.ClaimDeliveries(t.Context())
	require.NoError(t, err)
	results := []DeliveryResult{{ID: uuid.New(), Status: "delivered"}}
	require.NoError(t, store.AcknowledgeDeliveries(t.Context(), batch.Token, results))
	require.Equal(t, "claim", requests[0].Action)
	require.Equal(t, "ack", requests[1].Action)
	require.Equal(t, token, requests[1].Token)
	require.Equal(t, results, requests[1].Results)
}

func TestLambdaStoreRejectsFunctionErrorsAndInvalidResponses(t *testing.T) {
	for _, output := range []*lambda.InvokeOutput{
		nil,
		{StatusCode: 200, FunctionError: aws.String("Unhandled"), Payload: []byte(`{"errorMessage":"private details"}`)},
		{StatusCode: 202, Payload: []byte(`{}`)},
		{StatusCode: 200, Payload: []byte(`not json`)},
		{StatusCode: 200, Payload: []byte(`{}`)},
	} {
		store := NewLambdaStore(invokeFunc(func(context.Context, *lambda.InvokeInput, ...func(*lambda.Options)) (*lambda.InvokeOutput, error) {
			return output, nil
		}), "delivery")
		_, err := store.ClaimDeliveries(t.Context())
		require.Error(t, err)
		require.NotContains(t, err.Error(), "private details")
	}
}
