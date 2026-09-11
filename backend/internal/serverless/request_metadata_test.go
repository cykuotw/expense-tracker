package serverless

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"expense-tracker/backend/internal/observability"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/awslabs/aws-lambda-go-api-proxy/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWrapWithRequestMetadataAddsTrustedRuntimeIDs(t *testing.T) {
	var received observability.RuntimeRequestIDs
	handler := WrapWithRequestMetadata(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var ok bool
		received, ok = observability.RuntimeRequestIDsFromContext(r.Context())
		assert.True(t, ok)
		w.WriteHeader(http.StatusNoContent)
	}))

	ctx := lambdacontext.NewContext(context.Background(), &lambdacontext.LambdaContext{
		AwsRequestID: "lambda-request-id",
	})
	accessor := core.RequestAccessorV2{}
	request, err := accessor.EventToRequestWithContext(ctx, events.APIGatewayV2HTTPRequest{
		RawPath: "/health",
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			RequestID: "gateway-request-id",
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method: http.MethodGet,
				Path:   "/health",
			},
		},
	})
	require.NoError(t, err)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusNoContent, recorder.Code)
	assert.Equal(t, "gateway-request-id", received.APIGateway)
	assert.Equal(t, "lambda-request-id", received.AWSLambda)
}
