package serverless

import (
	"errors"
	"expense-tracker/backend/services/auth/google"
	"expense-tracker/backend/types"
	"log"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/awslabs/aws-lambda-go-api-proxy/core"
)

func WrapWithGoogleAuthorizerClaims(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if gatewayContext, ok := core.GetAPIGatewayV2ContextFromContext(r.Context()); ok {
			if claims, err := google.VerifiedClaimsFromAuthorizer(v2AuthorizerClaims(gatewayContext.Authorizer)); err == nil {
				r = r.WithContext(google.ContextWithVerifiedClaims(r.Context(), claims))
			} else if !errors.Is(err, types.ErrGoogleClaimsUnavailable) {
				log.Printf("failed to extract google authorizer claims: %v", err)
			}
		}

		next.ServeHTTP(w, r)
	})
}

func v2AuthorizerClaims(authorizer *events.APIGatewayV2HTTPRequestContextAuthorizerDescription) map[string]interface{} {
	if authorizer == nil {
		return nil
	}

	if authorizer.JWT != nil && len(authorizer.JWT.Claims) != 0 {
		return map[string]interface{}{
			"jwt": map[string]interface{}{
				"claims": authorizer.JWT.Claims,
			},
		}
	}

	if len(authorizer.Lambda) != 0 {
		return authorizer.Lambda
	}

	return nil
}
