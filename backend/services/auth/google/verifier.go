package google

import (
	"context"
	"expense-tracker/backend/config"
	"expense-tracker/backend/types"
	"log"
	"strings"

	"google.golang.org/api/idtoken"
)

var ValidateIDToken = func(ctx context.Context, rawToken string, audience string) (*idtoken.Payload, error) {
	return idtoken.Validate(ctx, rawToken, audience)
}

type Verifier interface {
	VerifyGoogleIDToken(ctx context.Context, rawToken string) (*types.VerifiedGoogleClaims, error)
}

type ClaimsVerifier struct {
	audience string
}

func NewClaimsVerifier() *ClaimsVerifier {
	return &ClaimsVerifier{
		audience: strings.TrimSpace(config.Envs.GoogleClientId),
	}
}

func (v *ClaimsVerifier) VerifyGoogleIDToken(ctx context.Context, rawToken string) (*types.VerifiedGoogleClaims, error) {
	payload, err := ValidateIDToken(ctx, rawToken, v.audience)
	if err != nil {
		return nil, types.ErrInvalidGoogleIDToken
	}

	if !isAllowedIssuer(payload.Issuer) {
		return nil, types.ErrInvalidGoogleIssuer
	}

	claims, err := normalizeVerifiedClaims(payload)
	if err != nil {
		return nil, err
	}

	log.Printf(
		"verified google claims: sub=%s email=%s email_verified=%t",
		claims.Subject,
		claims.Email,
		claims.EmailVerified != nil && *claims.EmailVerified,
	)

	return claims, nil
}

func isAllowedIssuer(issuer string) bool {
	switch strings.TrimSpace(issuer) {
	case "accounts.google.com", "https://accounts.google.com":
		return true
	default:
		return false
	}
}

func normalizeVerifiedClaims(payload *idtoken.Payload) (*types.VerifiedGoogleClaims, error) {
	if payload == nil {
		return nil, types.ErrMissingGoogleSubject
	}

	claims := make(map[string]interface{}, len(payload.Claims)+1)
	for key, value := range payload.Claims {
		claims[key] = value
	}
	claims["sub"] = payload.Subject

	return normalizeVerifiedClaimsMap(claims)
}

func normalizeVerifiedClaimsMap(claims map[string]interface{}) (*types.VerifiedGoogleClaims, error) {
	if claims == nil || stringClaim(claims, "sub") == "" {
		return nil, types.ErrMissingGoogleSubject
	}

	verifiedClaims := &types.VerifiedGoogleClaims{
		Subject:    stringClaim(claims, "sub"),
		Email:      stringClaim(claims, "email"),
		GivenName:  stringClaim(claims, "given_name"),
		FamilyName: stringClaim(claims, "family_name"),
		Name:       stringClaim(claims, "name"),
	}

	if emailVerified, ok, err := boolClaim(claims, "email_verified"); err != nil {
		return nil, types.ErrInvalidGoogleIDToken
	} else if ok {
		verifiedClaims.EmailVerified = &emailVerified
	}

	return verifiedClaims, nil
}

func stringClaim(claims map[string]interface{}, key string) string {
	if claims == nil {
		return ""
	}

	value, ok := claims[key]
	if !ok {
		return ""
	}

	stringValue, ok := value.(string)
	if !ok {
		return ""
	}

	return strings.TrimSpace(stringValue)
}

func boolClaim(claims map[string]interface{}, key string) (bool, bool, error) {
	if claims == nil {
		return false, false, nil
	}

	value, ok := claims[key]
	if !ok {
		return false, false, nil
	}

	switch typedValue := value.(type) {
	case bool:
		return typedValue, true, nil
	case string:
		switch strings.ToLower(strings.TrimSpace(typedValue)) {
		case "true":
			return true, true, nil
		case "false":
			return false, true, nil
		default:
			return false, false, types.ErrInvalidGoogleIDToken
		}
	default:
		return false, false, types.ErrInvalidGoogleIDToken
	}
}
