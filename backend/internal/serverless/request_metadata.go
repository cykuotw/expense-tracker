package serverless

import (
	"expense-tracker/backend/internal/observability"
	"net/http"

	"github.com/awslabs/aws-lambda-go-api-proxy/core"
)

// WrapWithRequestMetadata copies trusted serverless correlation identifiers
// into application context without exposing the rest of the gateway request.
func WrapWithRequestMetadata(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ids := observability.RuntimeRequestIDs{}
		if gatewayContext, ok := core.GetAPIGatewayV2ContextFromContext(r.Context()); ok {
			ids.APIGateway = gatewayContext.RequestID
		}
		if runtimeContext, ok := core.GetRuntimeContextFromContextV2(r.Context()); ok && runtimeContext != nil {
			ids.AWSLambda = runtimeContext.AwsRequestID
		}

		ctx := observability.ContextWithRuntimeRequestIDs(r.Context(), ids)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
