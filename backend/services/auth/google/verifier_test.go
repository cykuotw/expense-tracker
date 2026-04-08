package google

import (
	"context"
	"expense-tracker/backend/types"
	"fmt"
	"testing"

	"google.golang.org/api/idtoken"

	"github.com/stretchr/testify/assert"
)

func TestClaimsVerifier(t *testing.T) {
	originalValidate := ValidateIDToken
	defer func() {
		ValidateIDToken = originalValidate
	}()

	t.Run("normalizes verified claims", func(t *testing.T) {
		ValidateIDToken = func(ctx context.Context, rawToken string, audience string) (*idtoken.Payload, error) {
			assert.Equal(t, "raw-google-token", rawToken)
			assert.Equal(t, "test-client-id", audience)
			return &idtoken.Payload{
				Issuer:   "https://accounts.google.com",
				Audience: audience,
				Subject:  "google-sub-123",
				Claims: map[string]interface{}{
					"email":          "user@example.com",
					"email_verified": true,
					"given_name":     "Taylor",
					"family_name":    "Swift",
					"name":           "Taylor Swift",
				},
			}, nil
		}

		verifier := &ClaimsVerifier{audience: "test-client-id"}
		claims, err := verifier.VerifyGoogleIDToken(context.Background(), "raw-google-token")

		assert.NoError(t, err)
		assert.Equal(t, "google-sub-123", claims.Subject)
		assert.Equal(t, "user@example.com", claims.Email)
		if assert.NotNil(t, claims.EmailVerified) {
			assert.True(t, *claims.EmailVerified)
		}
		assert.Equal(t, "Taylor", claims.GivenName)
		assert.Equal(t, "Swift", claims.FamilyName)
		assert.Equal(t, "Taylor Swift", claims.Name)
	})

	t.Run("rejects invalid token", func(t *testing.T) {
		ValidateIDToken = func(ctx context.Context, rawToken string, audience string) (*idtoken.Payload, error) {
			return nil, fmt.Errorf("token invalid")
		}

		verifier := &ClaimsVerifier{audience: "test-client-id"}
		claims, err := verifier.VerifyGoogleIDToken(context.Background(), "raw-google-token")

		assert.Nil(t, claims)
		assert.ErrorIs(t, err, types.ErrInvalidGoogleIDToken)
	})

	t.Run("rejects invalid issuer", func(t *testing.T) {
		ValidateIDToken = func(ctx context.Context, rawToken string, audience string) (*idtoken.Payload, error) {
			return &idtoken.Payload{
				Issuer:  "https://malicious.example.com",
				Subject: "google-sub-123",
				Claims:  map[string]interface{}{},
			}, nil
		}

		verifier := &ClaimsVerifier{audience: "test-client-id"}
		claims, err := verifier.VerifyGoogleIDToken(context.Background(), "raw-google-token")

		assert.Nil(t, claims)
		assert.ErrorIs(t, err, types.ErrInvalidGoogleIssuer)
	})

	t.Run("rejects missing subject", func(t *testing.T) {
		ValidateIDToken = func(ctx context.Context, rawToken string, audience string) (*idtoken.Payload, error) {
			return &idtoken.Payload{
				Issuer: "https://accounts.google.com",
				Claims: map[string]interface{}{
					"email": "user@example.com",
				},
			}, nil
		}

		verifier := &ClaimsVerifier{audience: "test-client-id"}
		claims, err := verifier.VerifyGoogleIDToken(context.Background(), "raw-google-token")

		assert.Nil(t, claims)
		assert.ErrorIs(t, err, types.ErrMissingGoogleSubject)
	})
}
