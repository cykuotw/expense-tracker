package serverless

import (
	"context"
	"expense-tracker/backend/services/auth/google"
	"expense-tracker/backend/types"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/awslabs/aws-lambda-go-api-proxy/core"
	"github.com/stretchr/testify/assert"
)

func TestWrapWithGoogleAuthorizerClaims(t *testing.T) {
	t.Run("injects verified claims from jwt authorizer claims", func(t *testing.T) {
		var seen *types.VerifiedGoogleClaims
		handler := WrapWithGoogleAuthorizerClaims(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, err := google.VerifiedClaimsFromContext(r.Context())
			assert.NoError(t, err)
			seen = claims
			w.WriteHeader(http.StatusNoContent)
		}))

		req := mustAPIGatewayV2Request(t, &events.APIGatewayV2HTTPRequestContextAuthorizerDescription{
			JWT: &events.APIGatewayV2HTTPRequestContextAuthorizerJWTDescription{
				Claims: map[string]string{
					"sub":            "google-sub-123",
					"email":          "user@example.com",
					"email_verified": "true",
				},
			},
		})
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusNoContent, rr.Code)
		if assert.NotNil(t, seen) {
			assert.Equal(t, "google-sub-123", seen.Subject)
			assert.Equal(t, "user@example.com", seen.Email)
			if assert.NotNil(t, seen.EmailVerified) {
				assert.True(t, *seen.EmailVerified)
			}
		}
	})

	t.Run("injects verified claims from lambda authorizer claims", func(t *testing.T) {
		var seen *types.VerifiedGoogleClaims
		handler := WrapWithGoogleAuthorizerClaims(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, err := google.VerifiedClaimsFromContext(r.Context())
			assert.NoError(t, err)
			seen = claims
			w.WriteHeader(http.StatusNoContent)
		}))

		req := mustAPIGatewayV2Request(t, &events.APIGatewayV2HTTPRequestContextAuthorizerDescription{
			Lambda: map[string]interface{}{
				"sub":            "google-sub-456",
				"email_verified": true,
			},
		})
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusNoContent, rr.Code)
		if assert.NotNil(t, seen) {
			assert.Equal(t, "google-sub-456", seen.Subject)
			if assert.NotNil(t, seen.EmailVerified) {
				assert.True(t, *seen.EmailVerified)
			}
		}
	})

	t.Run("leaves context unchanged when claims are unavailable", func(t *testing.T) {
		handler := WrapWithGoogleAuthorizerClaims(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, err := google.VerifiedClaimsFromContext(r.Context())
			assert.Nil(t, claims)
			assert.ErrorIs(t, err, types.ErrGoogleClaimsUnavailable)
			w.WriteHeader(http.StatusNoContent)
		}))

		req := mustAPIGatewayV2Request(t, nil)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusNoContent, rr.Code)
	})
}

func mustAPIGatewayV2Request(t *testing.T, authorizer *events.APIGatewayV2HTTPRequestContextAuthorizerDescription) *http.Request {
	t.Helper()

	accessor := core.RequestAccessorV2{}
	req, err := accessor.EventToRequestWithContext(context.Background(), events.APIGatewayV2HTTPRequest{
		RawPath: "/auth/google/exchange",
		Headers: map[string]string{},
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			DomainName: "example.com",
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method:   http.MethodPost,
				Path:     "/auth/google/exchange",
				SourceIP: "127.0.0.1",
			},
			Authorizer: authorizer,
		},
	})
	if err != nil {
		t.Fatalf("failed to build api gateway request: %v", err)
	}

	return req
}
