package google

import (
	"expense-tracker/backend/types"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVerifiedClaimsFromAuthorizer(t *testing.T) {
	t.Run("extracts nested claims map", func(t *testing.T) {
		claims, err := VerifiedClaimsFromAuthorizer(map[string]interface{}{
			"claims": map[string]interface{}{
				"sub":            "google-sub-123",
				"email":          "user@example.com",
				"email_verified": "true",
				"given_name":     "Taylor",
				"family_name":    "Swift",
				"name":           "Taylor Swift",
			},
		})

		assert.NoError(t, err)
		assert.Equal(t, "google-sub-123", claims.Subject)
		assert.Equal(t, "user@example.com", claims.Email)
		assert.Equal(t, "Taylor", claims.GivenName)
		assert.Equal(t, "Swift", claims.FamilyName)
		assert.Equal(t, "Taylor Swift", claims.Name)
		if assert.NotNil(t, claims.EmailVerified) {
			assert.True(t, *claims.EmailVerified)
		}
	})

	t.Run("extracts flat claims map", func(t *testing.T) {
		claims, err := VerifiedClaimsFromAuthorizer(map[string]interface{}{
			"sub":            "google-sub-456",
			"email":          "flat@example.com",
			"email_verified": true,
		})

		assert.NoError(t, err)
		assert.Equal(t, "google-sub-456", claims.Subject)
		assert.Equal(t, "flat@example.com", claims.Email)
		if assert.NotNil(t, claims.EmailVerified) {
			assert.True(t, *claims.EmailVerified)
		}
	})

	t.Run("extracts nested jwt claims map", func(t *testing.T) {
		claims, err := VerifiedClaimsFromAuthorizer(map[string]interface{}{
			"jwt": map[string]interface{}{
				"claims": map[string]interface{}{
					"sub":            "google-sub-789",
					"email_verified": "false",
				},
			},
		})

		assert.NoError(t, err)
		assert.Equal(t, "google-sub-789", claims.Subject)
		if assert.NotNil(t, claims.EmailVerified) {
			assert.False(t, *claims.EmailVerified)
		}
	})

	t.Run("returns unavailable when claims are missing", func(t *testing.T) {
		claims, err := VerifiedClaimsFromAuthorizer(map[string]interface{}{})

		assert.Nil(t, claims)
		assert.ErrorIs(t, err, types.ErrGoogleClaimsUnavailable)
	})

	t.Run("returns invalid token when boolean claim is malformed", func(t *testing.T) {
		claims, err := VerifiedClaimsFromAuthorizer(map[string]interface{}{
			"claims": map[string]interface{}{
				"sub":            "google-sub-123",
				"email_verified": "definitely",
			},
		})

		assert.Nil(t, claims)
		assert.ErrorIs(t, err, types.ErrInvalidGoogleIDToken)
	})
}
