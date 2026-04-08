package google

import "expense-tracker/backend/types"

func VerifiedClaimsFromAuthorizer(authorizer map[string]interface{}) (*types.VerifiedGoogleClaims, error) {
	claims, ok := authorizerClaims(authorizer)
	if !ok {
		return nil, types.ErrGoogleClaimsUnavailable
	}

	return normalizeVerifiedClaimsMap(claims)
}

func authorizerClaims(authorizer map[string]interface{}) (map[string]interface{}, bool) {
	if claims, ok := claimsMap(authorizer["claims"]); ok {
		return claims, true
	}

	if jwt, ok := claimsMap(authorizer["jwt"]); ok {
		if claims, ok := claimsMap(jwt["claims"]); ok {
			return claims, true
		}
	}

	if claims, ok := claimsMap(authorizer); ok && stringClaim(claims, "sub") != "" {
		return claims, true
	}

	return nil, false
}

func claimsMap(value interface{}) (map[string]interface{}, bool) {
	switch typed := value.(type) {
	case map[string]interface{}:
		return typed, len(typed) != 0
	case map[string]string:
		if len(typed) == 0 {
			return nil, false
		}

		claims := make(map[string]interface{}, len(typed))
		for key, value := range typed {
			claims[key] = value
		}
		return claims, true
	default:
		return nil, false
	}
}
